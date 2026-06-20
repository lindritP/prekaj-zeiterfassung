#!/usr/bin/env bash
# ============================================================================
# BUG-01 — end_zeit in der Zukunft wird akzeptiert
# Anwendungsfall: Zeiterfassung beenden / korrigieren
# Erwartet : 400 (analog zur vorhandenen Zukunfts-Pruefung von start_zeit)
# Tatsaechl.: 200 — absurd grosse Dauer wird gespeichert (POST /stop UND PATCH)
# Ein-Befehl-Repro fuer die Bildschirmaufzeichnung:  bash BUG-01-future-end-zeit.sh
# ============================================================================
BASE="${BASE:-http://localhost:8080/api/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@fliesenprekaj.local}"
ADMIN_PASS="${ADMIN_PASS:-change-me-locally}"
J='Content-Type: application/json'
tmp=$(mktemp)

echo "############################################################"
echo "# BUG-01  end_zeit in der Zukunft wird akzeptiert"
echo "# Erwartet: 400  |  Tatsaechlich: 200"
echo "############################################################"
echo

AT=$(curl -s -X POST "$BASE/auth/login" -H "$J" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"passwort\":\"$ADMIN_PASS\",\"client\":\"web\"}" | jq -r .access_token)
EM="bug01-$(date +%s)-$RANDOM@test.local"
curl -s -X POST "$BASE/arbeiter" -H "Authorization: Bearer $AT" -H "$J" \
  -d "{\"name\":\"BUG01 Worker\",\"email\":\"$EM\",\"passwort\":\"test1234\",\"rolle\":\"arbeiter\"}" >/dev/null
WT=$(curl -s -X POST "$BASE/auth/login" -H "$J" \
  -d "{\"email\":\"$EM\",\"passwort\":\"test1234\",\"client\":\"web\"}" | jq -r .access_token)

echo ">>> (Variante A: STOP) POST /zeit/start  {\"start_zeit\":\"2026-06-20T10:00:00Z\"}"
curl -s -X POST "$BASE/zeit/start" -H "Authorization: Bearer $WT" -H "$J" \
  -d '{"start_zeit":"2026-06-20T10:00:00Z"}' | jq -c '{id,start_zeit,end_zeit}'
echo
echo ">>> POST /zeit/stop  {\"end_zeit\":\"2030-01-01T00:00:00Z\"}   (~3,5 Jahre in der Zukunft)"
curl -s -X POST "$BASE/zeit/stop" -H "Authorization: Bearer $WT" -H "$J" \
  -d '{"end_zeit":"2030-01-01T00:00:00Z"}' -o "$tmp" -w "    HTTP-Status: %{http_code}\n"
echo "    Antwort: $(jq -c '{end_zeit,dauer_minuten}' "$tmp")"
echo "    => HTTP 200 statt 400; dauer_minuten ~ $(jq -r '.dauer_minuten' "$tmp") (=$(jq -r '(.dauer_minuten/1440)|floor' "$tmp") Tage) => FEHLER"
echo

echo ">>> (Variante B: PATCH) POST /zeit/start  {\"start_zeit\":\"2026-06-20T11:00:00Z\"}"
ZID=$(curl -s -X POST "$BASE/zeit/start" -H "Authorization: Bearer $WT" -H "$J" \
  -d '{"start_zeit":"2026-06-20T11:00:00Z"}' | jq -r .id)
echo "    laufende Buchung id=$ZID"
echo ">>> PATCH /zeit/$ZID  {\"end_zeit\":\"2030-01-01T00:00:00Z\"}"
curl -s -X PATCH "$BASE/zeit/$ZID" -H "Authorization: Bearer $WT" -H "$J" \
  -d '{"end_zeit":"2030-01-01T00:00:00Z"}' -o "$tmp" -w "    HTTP-Status: %{http_code}\n"
echo "    Antwort: $(jq -c '{end_zeit,dauer_minuten}' "$tmp")"
echo "    => auch via PATCH akzeptiert (HTTP 200) => FEHLER"
rm -f "$tmp"
