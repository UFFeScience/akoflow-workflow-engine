started_at=$(date +%s 2>/dev/null || printf '0')
root=${AKOFLOW_OBSERVATION_ROOT:-.}
initial_files=$(find "$root" -type f 2>/dev/null | wc -l | tr -d ' ')
initial_bytes=$(find "$root" -type f -exec wc -c {} + 2>/dev/null | awk 'END { print $1+0 }')

"$@"
activity_exit_code=$?

finished_at=$(date +%s 2>/dev/null || printf '0')
final_files=$(find "$root" -type f 2>/dev/null | wc -l | tr -d ' ')
final_bytes=$(find "$root" -type f -exec wc -c {} + 2>/dev/null | awk 'END { print $1+0 }')
output_bytes=$((final_bytes > initial_bytes ? final_bytes - initial_bytes : 0))
duration=$((finished_at >= started_at ? finished_at - started_at : 0))

manifest_format='{"schemaVersion":1,"runId":"%s","activityId":"%s","attempt":1,'
manifest_format=$manifest_format'"runtime":"kubernetes","root":"%s","startedAt":%s,'
manifest_format=$manifest_format'"finishedAt":%s,"exitCode":%s,"files":[],"phases":['
manifest_format=$manifest_format'{"phase":"execution","status":"%s","startedAt":%s,'
manifest_format=$manifest_format'"finishedAt":%s,"durationSeconds":%s}],"summary":{'
manifest_format=$manifest_format'"initialFiles":%s,"finalFiles":%s,"createdFiles":0,'
manifest_format=$manifest_format'"modifiedFiles":0,"deletedFiles":0,"outputBytes":%s}}'
manifest=$(printf "$manifest_format" \
  "$AKOFLOW_RUN_ID" "$AKOFLOW_ACTIVITY_ID" "$root" "$started_at" "$finished_at" "$activity_exit_code" \
  "$(if [ "$activity_exit_code" -eq 0 ]; then printf completed; else printf failed; fi)" \
  "$started_at" "$finished_at" "$duration" "${initial_files:-0}" "${final_files:-0}" "$output_bytes")

if command -v base64 >/dev/null 2>&1; then
  encoded_manifest=$(printf '%s' "$manifest" | base64 | tr -d '\n')
  printf '\n__AKOFLOW_MANIFEST_PREFIX__%s\n' "$encoded_manifest"
else
  printf '\nAKOFLOW_OBSERVATION_ERROR=base64 utility is unavailable; activity result was preserved\n' >&2
fi

exit "$activity_exit_code"
