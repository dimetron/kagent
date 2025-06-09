#!/bin/bash

DEST_DIR=$(dirname "$0")
BASE_URL=https://gitlab.corp.amdocs.com/ansible-roles/setup-os-repositories-and-ca/raw/master/files/ca/

HTTP_PROXY=
HTTPS_PROXY=
NO_PROXY=*

mkdir -p $DEST_DIR
for cert in 1_Amdocs_Root_CA.crt 2_Amdocs_Class3_CA.crt 3_Amdocs_SSL_Proxy_CA.crt 4_Amdocs_Certificate_Enrollment_CA.crt 5_Fortinet_proxy_CA.crt 6_Amdocs_Root_CA_2020_2045.crt 7_Amdocs_AmdocsRSARootCA_2023_2043.crt 8_Amdocs_CorpRSASubCA_2023_2033.crt FullAmdocsCA.crt
do
  echo "downloading $BASE_URL/$cert"
  if [[ ! -r $DEST_DIR/$cert ]]; then
    wget --tries=1 --timeout=3 -N -c --quiet --no-check-certificate -O $DEST_DIR/$cert $BASE_URL/$cert
  else
    echo "$DEST_DIR/$cert already exists - skipping"
  fi
done
ls $DEST_DIR
