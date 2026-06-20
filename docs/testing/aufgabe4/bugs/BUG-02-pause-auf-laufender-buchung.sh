#!/usr/bin/env bash
# ============================================================================
# BUG-02 — pause_minuten auf laufender (noch nicht beendeter) Buchung
# Anwendungsfall: Zeiterfassung korrigieren (PATCH)
# Erwartet : 400 — eine Pause ist ohne Spanne (end_zeit fehlt) nicht pruefbar
#            (Regel "pause <= Spanne") und fachlich erst nach dem Stop sinnvoll.
# Tatsaechl.: 200 — Wert wird gesetzt/bestaetigt, aber beim Stop still verworfen.
# Ein-Befehl-Repro fuer die Bildschirmaufzeichnung:  bash BUG-02-pause-auf-laufender-buchung.sh
# ============================================================================
BASE="${BASE:-http://localhost:8080/api/v1}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@fliesenprekaj.local}"
ADMIN_PASS="${ADMIN_PASS:-change-me-locally}"
J='Content-Type: application/json'
tmp=$(mktemp)

echo "############################################################"
echo "# BUG-02  Pause auf laufender Buchung wird akzeptiert"
echo "# Erwartet: 400  |  Tatsaechlich: 200 (Wert beim Stop verworfen)"
echo "############################################################"
echo

AT=$(curl -s -X POST "$BASE/auth/login" -H "$J" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"passwort\":\"$ADMIN_PASS\",\"client\":\"web\"}" | jq -r .access_token)
EM="bug02-$(date +%s)-$RANDOM@test.local"
curl -s -X POST "$BASE/arbeiter" -H "Authorization: Bearer $AT" -H "$J" \
  -d "{\"name\":\"BUG02 Worker\",\"email\":\"$EM\",\"passwort\":\"test1234\",\"rolle\":\"arbeiter\"}" >/dev/null
WT=$(curl -s -X POST "$BASE/auth/login" -H "$J" \
  -d "{\"email\":\"$EM\",\"passwort\":\"test1234\",\"client\":\"web\"}" | jq -r .access_token)

echo ">>> POST /zeit/start  {}  (laufende Buchung, end_zeit=null)"
ZID=$(curl -s -X POST "$BASE/zeit/start" -H "Authorization: Bearer $WT" -H "$J" -d '{}' | jq -r .id)
echo "    laufende Buchung id=$ZID"
echo
echo ">>> PATCH /zeit/$ZID  {\"pause_minuten\":30}   (Buchung laeuft noch)"
curl -s -X PATCH "$BASE/zeit/$ZID" -H "Authorization: Bearer $WT" -H "$J" \
  -d '{"pause_minuten":30}' -o "$tmp" -w "    HTTP-Status: %{http_code}\n"
echo "    Antwort: $(jq -c '{end_zeit,pause_minuten,dauer_minuten}' "$tmp")"
echo "    => HTTP 200, pause_minuten=30 auf laufender Buchung gesetzt (erwartet 400) => FEHLER"
echo
echo ">>> POST /zeit/stop {\"end_zeit\":\"...+1h\"}  -> Pause wird neu berechnet/verworfen"
ST=$(curl -s -X GET "$BASE/zeit/laufend" -H "Authorization: Bearer $WT" | jq -r .start_zeit)
END=$(python3 -c "import datetime as d;print((d.datetime.fromisoformat('${ST}'.replace('Z','+00:00'))+d.timedelta(hours=1)).strftime('%Y-%m-%dT%H:%M:%SZ'))")
curl -s -X POST "$BASE/zeit/stop" -H "Authorization: Bearer $WT" -H "$J" \
  -d "{\"end_zeit\":\"$END\"}" -o "$tmp" -w "    HTTP-Status: %{http_code}\n"
echo "    Antwort: $(jq -c '{pause_minuten,dauer_minuten}' "$tmp")"
echo "    => zuvor gesetzte Pause=30 ist beim Stop verschwunden (still verworfen)"
rm -f "$tmp"
