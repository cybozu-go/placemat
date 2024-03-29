#!/bin/sh -ex

SUITE=$1
TARGET=$2

. $(dirname $0)/env

if [ -z "${INSTANCE_NAME}" ]; then
  echo 'Please set $INSTANCE_NAME'
  exit 1
fi

delete_instance() {
  if [ $RET -ne 0 ]; then
    # do not delete GCP instance upon test failure to help debugging.
    return
  fi
  $GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
}

# Create GCE instance
$GCLOUD compute instances delete ${INSTANCE_NAME} --zone ${ZONE} || true
$GCLOUD compute instances create ${INSTANCE_NAME} \
  --zone ${ZONE} \
  --machine-type ${MACHINE_TYPE} \
  --image vmx-enabled \
  --boot-disk-type ${DISK_TYPE} \
  --boot-disk-size ${BOOT_DISK_SIZE} \
  --local-ssd interface=scsi

RET=0
trap delete_instance INT QUIT TERM 0

# Run multi-host test
for i in $(seq 300); do
  if $GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command=date 2>/dev/null; then
    break
  fi
  sleep 1
done

cat >run.sh <<EOF
#!/bin/sh -e

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch

# Run mtest
GOPATH=\$HOME/go
export GOPATH
GO111MODULE=on
export GO111MODULE
PATH=/usr/local/go/bin:\$GOPATH/bin:\$PATH
export PATH

git clone https://github.com/${GITHUB_REPOSITORY} \
    \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
cd \$HOME/go/src/github.com/${GITHUB_REPOSITORY}
if [ "${GITHUB_EVENT_NAME}" = "pull_request" ]; then
  git fetch --prune origin +${GITHUB_SHA}:refs/remotes/pull/merge
fi
git checkout -qf ${GITHUB_SHA}

cd v2/e2e
cp /assets/ubuntu-*.img .
exec make test
EOF
chmod +x run.sh

$GCLOUD compute scp --zone=${ZONE} run.sh cybozu@${INSTANCE_NAME}:
set +e
$GCLOUD compute ssh --zone=${ZONE} cybozu@${INSTANCE_NAME} --command='sudo /home/cybozu/run.sh'
RET=$?

exit $RET
