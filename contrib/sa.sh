#!/bin/bash
SERVICE_ACCOUNT_NAME=nodalingresser
SERVICE_ACCOUNT_DEST=.private/$SERVICE_ACCOUNT_NAME.json
PROJECT=$1

if [[ "$PROJECT" == "" ]]; then
  echo "A project must be provided."
  exit 1
fi

gcloud --project $PROJECT iam service-accounts create \
    $SERVICE_ACCOUNT_NAME \
    --display-name $SERVICE_ACCOUNT_NAME

SA_EMAIL=$(gcloud --project $PROJECT iam service-accounts list \
    --filter="displayName:$SERVICE_ACCOUNT_NAME" \
    --format='value(email)')

gcloud projects add-iam-policy-binding $PROJECT \
    --member serviceAccount:$SA_EMAIL \
    --role roles/dns.admin

mkdir -p $(dirname $SERVICE_ACCOUNT_DEST)

gcloud --project $PROJECT iam service-accounts keys create $SERVICE_ACCOUNT_DEST \
    --iam-account $SA_EMAIL
