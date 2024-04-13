#!/usr/bin/env bash
set -Eeo pipefail

dependencies=(awk curl gzip jq)
for program in "${dependencies[@]}"; do
    command -v "$program" >/dev/null 2>&1 || {
        echo >&2 "Couldn't find dependency: $program. Aborting."
        exit 1
    }
done

if [[ "${RUNNING_IN_DOCKER}" ]]; then
    source "/app/aemet_exporter.conf"
else
    # shellcheck source=/dev/null
    source "$CREDENTIALS_DIRECTORY/creds"
fi

[[ -z "${INFLUXDB_HOST}" ]] && echo >&2 "INFLUXDB_HOST is empty. Aborting" && exit 1
[[ -z "${INFLUXDB_API_TOKEN}" ]] && echo >&2 "INFLUXDB_API_TOKEN is empty. Aborting" && exit 1
[[ -z "${ORG}" ]] && echo >&2 "ORG is empty. Aborting" && exit 1
[[ -z "${BUCKET}" ]] && echo >&2 "BUCKET is empty. Aborting" && exit 1
[[ -z "${AEMET_API_KEY}" ]] && echo >&2 "AEMET_API_KEY is empty. Aborting" && exit 1
[[ -z "${AEMET_WEATHER_STATION_CODE}" ]] && echo >&2 "AEMET_WEATHER_STATION_CODE is empty. Aborting" && exit 1

AWK=$(command -v awk)
CURL=$(command -v curl)
GZIP=$(command -v gzip)
JQ=$(command -v jq)

INFLUXDB_URL="https://$INFLUXDB_HOST/api/v2/write?precision=s&org=$ORG&bucket=$BUCKET"
AEMET_WEATHER_URL="https://opendata.aemet.es/opendata/api/observacion/convencional/datos/estacion/$AEMET_WEATHER_STATION_CODE"

aemet_weather_redirect_url=$(
    $CURL --silent --fail --show-error --compressed \
        --header "api_key: $AEMET_API_KEY" \
        "$AEMET_WEATHER_URL" |
        $JQ --raw-output '.datos'
)

aemet_weather_json=$($CURL --silent --fail --show-error --compressed "$aemet_weather_redirect_url")

weather_stats=$(
    echo "$aemet_weather_json" |
        $JQ --raw-output "
        (.[] |
        [\"${AEMET_WEATHER_STATION_CODE}\",
        if has(\"ta\") then .ta else 0 end,
        if has(\"hr\") then .hr else 0 end,
        if has(\"pres\") then .pres else 0 end,
        if has(\"vv\") then .vv else 0 end,
        if has(\"dv\") then .dv else 0 end,
        if has(\"vmax\") then .vmax else 0 end,
        if has(\"prec\") then .prec else 0 end,
        if has(\"tpr\") then .tpr else 0 end,
        if has(\"vis\") then .vis else 0 end,
        if has(\"inso\") then .inso else 0 end,
        if has(\"nieve\") then .nieve else 0 end,
        (.fint[0:18] + \"Z\" | fromdate)
        ])
        | @tsv" |
        $AWK '{printf "aemet_weather_conditions,station=%s temperature=%s,humidity=%s,pressure=%s,windspeed=%s,winddirection=%s,windgust=%s,precipitation=%s,dewpoint=%s,visibility=%s,insolation=%s,snow=%s %s\n", $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13}'
)

echo "$weather_stats" | $GZIP |
    $CURL --silent --fail --show-error \
        --request POST "${INFLUXDB_URL}" \
        --header 'Content-Encoding: gzip' \
        --header "Authorization: Token $INFLUXDB_API_TOKEN" \
        --header "Content-Type: text/plain; charset=utf-8" \
        --header "Accept: application/json" \
        --data-binary @-
