#!/bin/bash

CREDS_FILE="${HOME}/.aws/credentials"

echo '-=-= Examining profiles:'
echo '  username: aws --profile ${p} iam get-user 2>/dev/null | jq -r .User.UserName'
echo '  accountName: aws --profile ${p} iam list-account-aliases 2>/dev/null | jq -r .AccountAliases[0]'
echo '  accountId: aws --profile ${p} sts get-caller-identity | jq -r .Account'
echo '  keyId: aws --profile ${p} configure get aws_access_key_id'
echo '  lastUsed: aws --profile ${p} iam get-access-key-last-used --access-key-id ${keyId} | jq -r .AccessKeyLastUsed.LastUsedDate)'
echo '  keys created/status: aws --profile ${p} iam list-access-keys | jq ...'

echo 'shown as:'
echo '  profile: username@accountName (accountId)'
echo '    THISKEY - lastUsed'
echo '    OTHERKEYS - (status) created at'
echo
profiles=$(grep '^\[.*\]' ${CREDS_FILE}| tr -d '[]')

# echo ${profiles}
for p in ${profiles}; do
  username=$(aws --profile ${p} iam get-user 2>/dev/null | jq -r .User.UserName)
  [[ -z  ${username}  ]] && username='anonymous'
  accountName=$(aws --profile ${p} iam list-account-aliases 2>/dev/null | jq -r .AccountAliases[0])
  [[ -z  ${accountName}  ]] && accountName='noalias'
  accountId=$(aws --profile ${p} sts get-caller-identity | jq -r .Account)
  echo " ${p}: ${username}@${accountName} (${accountId})"

  # Confusing since lastUsed will include chacking from this script!!!

  # The key used for this profile (from local config)
  keyId=$(aws --profile ${p} configure get aws_access_key_id)
  # keyIds=`aws iam list-access-keys --user-name XXX | jq -r .AccessKeyMetadata[].AccessKeyId`
  lastUsed=$(aws --profile ${p} iam get-access-key-last-used --access-key-id ${keyId} 2>/dev/null| jq -r .AccessKeyLastUsed.LastUsedDate)
  [[ -z  ${lastUsed}  ]] && lastUsed='Unknown'
  echo "    ${keyId} last used ${lastUsed}"


  aws --profile ${p} iam list-access-keys 2>/dev/null | jq -r '.AccessKeyMetadata[] | "    "+.AccessKeyId+" ("+.Status+") created at " +.CreateDate'

done
echo