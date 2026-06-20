#!/usr/bin/env python3
"""
Reproduzierbarer Generator für die Postman-Collection zu Aufgabe 4
(Anwendungsfall „Zeiterfassung starten/beenden", spezifikationsbasierte Testfälle).

Aufruf:
    python3 build-collection.py
erzeugt:
    zeiterfassung.postman_collection.json   (Collection v2.1, von Newman ausführbar)

Die Collection legt im Setup-Ordner Testdaten an (Admin-Login, Baustelle, zwei
Worker A/B) und führt anschließend die Testfälle in fachlich sinnvoller Reihenfolge
aus. Hilfs-/Reset-Requests (Präfix „(setup)" / „(reset)") stellen nur den Zustand her
und sind keine Testfälle. Die beiden Requests mit „DEF-…" im Namen prüfen das
spec-erwartete Verhalten und schlagen bewusst fehl (= gefundene Abweichungen).
"""
import json

COLLECTION_VARS = [
    "adminToken", "tokenA", "tokenB", "baustelleId", "emailA", "emailB",
    "workerAId", "workerBId", "zidHappy", "zid360", "zidB", "zidRun",
    "notiz1000", "notiz1001",
]

# Collection-weiter Pre-request: einmalig eindeutige E-Mails + Notiz-Strings setzen.
COLLECTION_PREREQ = [
    "if (!pm.collectionVariables.get('emailA')) pm.collectionVariables.set('emailA', 'wa-' + Date.now() + '-' + Math.floor(Math.random()*1e6) + '@test.local');",
    "if (!pm.collectionVariables.get('emailB')) pm.collectionVariables.set('emailB', 'wb-' + Date.now() + '-' + Math.floor(Math.random()*1e6) + '@test.local');",
    "if (!pm.collectionVariables.get('notiz1000')) pm.collectionVariables.set('notiz1000', 'x'.repeat(1000));",
    "if (!pm.collectionVariables.get('notiz1001')) pm.collectionVariables.set('notiz1001', 'x'.repeat(1001));",
]


def hdr(auth=None, with_ct=True):
    h = []
    if with_ct:
        h.append({"key": "Content-Type", "value": "application/json"})
    tok = {"admin": "{{adminToken}}", "A": "{{tokenA}}", "B": "{{tokenB}}"}.get(auth)
    if tok:
        h.append({"key": "Authorization", "value": "Bearer " + tok})
    return h


def url(segs, query=None):
    raw = "{{baseUrl}}/" + "/".join(segs)
    u = {"raw": raw, "host": ["{{baseUrl}}"], "path": segs}
    if query:
        u["query"] = [{"key": k, "value": v} for k, v in query]
        u["raw"] = raw + "?" + "&".join(k + "=" + v for k, v in query)
    return u


def body(obj):
    return {"mode": "raw", "raw": json.dumps(obj, ensure_ascii=False),
            "options": {"raw": {"language": "json"}}}


def req(name, method, segs, auth=None, b=None, query=None, tests=None, no_ct=False):
    r = {"method": method, "header": hdr(auth, with_ct=not no_ct), "url": url(segs, query)}
    if b is not None:
        r["body"] = body(b)
    item = {"name": name, "request": r}
    if tests:
        item["event"] = [{"listen": "test",
                          "script": {"type": "text/javascript", "exec": tests}}]
    return item


def folder(name, items):
    return {"name": name, "item": items}


# ── Assertion-Bausteine (JS-Zeilen) ─────────────────────────────────────────
def st(code, label=None):
    lbl = label or ("Status " + str(code))
    return ["pm.test(" + json.dumps(lbl) + ", function () { pm.response.to.have.status(" + str(code) + "); });"]


def errcode(code):
    return ["pm.test('error.code == " + code + "', function () { pm.expect(pm.response.json().error.code).to.eql('" + code + "'); });"]


def setvar(var, expr):
    return ["pm.collectionVariables.set('" + var + "', " + expr + ");"]


# ── Collection-Items ────────────────────────────────────────────────────────
setup = folder("0 — Setup (Testdaten)", [
    req("Login Admin", "POST", ["auth", "login"],
        b={"email": "{{adminEmail}}", "passwort": "{{adminPassword}}", "client": "web"},
        tests=st(200) + setvar("adminToken", "pm.response.json().access_token")),
    req("(setup) Baustelle anlegen", "POST", ["baustellen"], auth="admin",
        b={"name": "A4 Test-Baustelle", "adresse": "Teststraße 1"},
        tests=["pm.test('2xx', function () { pm.expect(pm.response.code).to.be.oneOf([200,201]); });"]
              + setvar("baustelleId", "pm.response.json().id")),
    req("(setup) Worker A anlegen", "POST", ["arbeiter"], auth="admin",
        b={"name": "Test Worker A", "email": "{{emailA}}", "passwort": "test1234",
           "rolle": "arbeiter", "wochenstunden": "40"},
        tests=["pm.test('2xx', function () { pm.expect(pm.response.code).to.be.oneOf([200,201]); });"]
              + setvar("workerAId", "pm.response.json().id")),
    req("(setup) Worker B anlegen", "POST", ["arbeiter"], auth="admin",
        b={"name": "Test Worker B", "email": "{{emailB}}", "passwort": "test1234",
           "rolle": "arbeiter", "wochenstunden": "40"},
        tests=["pm.test('2xx', function () { pm.expect(pm.response.code).to.be.oneOf([200,201]); });"]
              + setvar("workerBId", "pm.response.json().id")),
    req("(setup) Login Worker A", "POST", ["auth", "login"],
        b={"email": "{{emailA}}", "passwort": "test1234", "client": "web"},
        tests=st(200) + setvar("tokenA", "pm.response.json().access_token")),
    req("(setup) Login Worker B", "POST", ["auth", "login"],
        b={"email": "{{emailB}}", "passwort": "test1234", "client": "web"},
        tests=st(200) + setvar("tokenB", "pm.response.json().access_token")),
])

reset_A = lambda label="(reset) laufende Buchung A stoppen": req(
    label, "POST", ["zeit", "stop"], auth="A", b={},
    tests=["pm.test('reset ok', function () { pm.expect(pm.response.code).to.be.oneOf([200,409]); });"])

start_A = lambda sz, label="(setup) Buchung A starten": req(
    label, "POST", ["zeit", "start"], auth="A", b={"start_zeit": sz},
    tests=["pm.test('start ok/bereits laufend', function () { pm.expect(pm.response.code).to.be.oneOf([201,409]); });"])

f_start = folder("1 — Start (POST /zeit/start)", [
    req("TC-START-01 Start (Hauptpfad, Baustelle + Vergangenheit + Notiz) -> 201", "POST",
        ["zeit", "start"], auth="A",
        b={"baustelle_id": "{{baustelleId}}", "start_zeit": "2026-06-20T08:00:00Z", "notiz": "Fliesen Bad"},
        tests=st(201) + [
            "pm.test('end_zeit ist null (laufend)', function () { pm.expect(pm.response.json().end_zeit).to.eql(null); });",
            "pm.test('baustelle_id gesetzt', function () { pm.expect(pm.response.json().baustelle_id).to.eql(pm.collectionVariables.get('baustelleId')); });",
        ] + setvar("zidHappy", "pm.response.json().id")),
    req("TC-START-02 Start während laufender Buchung -> 409", "POST", ["zeit", "start"], auth="A",
        b={}, tests=st(409) + errcode("running_exists")),
    reset_A(),
    req("TC-START-03 Start ohne baustelle_id (optional) -> 201", "POST", ["zeit", "start"], auth="A",
        b={"start_zeit": "2026-06-20T09:00:00Z"},
        tests=st(201) + ["pm.test('baustelle_id null', function () { pm.expect(pm.response.json().baustelle_id).to.eql(null); });"]),
    reset_A(),
    req("TC-START-04 Start mit unbekannter baustelle_id -> 400", "POST", ["zeit", "start"], auth="A",
        b={"baustelle_id": "019ee601-0000-7000-8000-000000000000"},
        tests=st(400) + errcode("bad_request")),
    req("TC-START-05 Start start_zeit in der Zukunft -> 400", "POST", ["zeit", "start"], auth="A",
        b={"start_zeit": "2027-01-01T00:00:00Z"},
        tests=st(400) + errcode("bad_request") + [
            "pm.test('Meldung nennt Zukunft', function () { pm.expect(pm.response.json().error.message).to.include('Zukunft'); });"]),
    req("TC-START-06 Notiz = 1000 Zeichen (Grenze gueltig) -> 201", "POST", ["zeit", "start"], auth="A",
        b={"notiz": "{{notiz1000}}"}, tests=st(201)),
    reset_A(),
    req("TC-START-07 Notiz = 1001 Zeichen (Grenze ungueltig) -> 400", "POST", ["zeit", "start"], auth="A",
        b={"notiz": "{{notiz1001}}"}, tests=st(400) + errcode("validation_error")),
    req("TC-START-08 start_zeit mit +02:00-Offset -> 201, als UTC gespeichert", "POST", ["zeit", "start"], auth="A",
        b={"start_zeit": "2026-06-19T10:00:00+02:00"},
        tests=st(201) + ["pm.test('als UTC (08:00:00Z) normalisiert', function () { pm.expect(pm.response.json().start_zeit).to.eql('2026-06-19T08:00:00Z'); });"]),
    reset_A(),
])

f_stop = folder("2 — Stop (POST /zeit/stop)", [
    start_A("2026-06-20T10:00:00Z", "(setup) Buchung A starten (10:00)"),
    req("TC-STOP-01 Stop (Hauptpfad, end 12:30, Spanne 150) -> 200", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2026-06-20T12:30:00Z"},
        tests=st(200) + [
            "pm.test('dauer_minuten == 150', function () { pm.expect(pm.response.json().dauer_minuten).to.eql(150); });",
            "pm.test('pause_minuten == 0 (Spanne < 6h)', function () { pm.expect(pm.response.json().pause_minuten).to.eql(0); });",
        ]),
    req("TC-STOP-02 Stop ohne laufende Buchung -> 409", "POST", ["zeit", "stop"], auth="A",
        b={}, tests=st(409) + errcode("no_running")),
    start_A("2026-06-20T10:00:00Z", "(setup) Buchung A starten (10:00)"),
    req("TC-STOP-03 Stop end_zeit == start_zeit -> 400", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2026-06-20T10:00:00Z"},
        tests=st(400) + errcode("bad_request")),
    req("TC-STOP-04 Stop end_zeit < start_zeit -> 400", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2026-06-20T09:00:00Z"},
        tests=st(400) + errcode("bad_request")),
    reset_A(),
    start_A("2026-06-20T08:00:00Z", "(setup) Buchung A starten (08:00) fuer Spanne 360"),
    req("TC-STOP-05 Spanne genau 360 min (08:00-14:00) -> Pause 0, Dauer 360", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2026-06-20T14:00:00Z"},
        tests=st(200) + [
            "pm.test('pause_minuten == 0', function () { pm.expect(pm.response.json().pause_minuten).to.eql(0); });",
            "pm.test('dauer_minuten == 360', function () { pm.expect(pm.response.json().dauer_minuten).to.eql(360); });",
        ] + setvar("zid360", "pm.response.json().id")),
    start_A("2026-06-20T08:00:00Z", "(setup) Buchung A starten (08:00) fuer Spanne 361"),
    req("TC-STOP-06 Spanne 361 min (08:00-14:01) -> gesetzl. Pause 30, Dauer 331", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2026-06-20T14:01:00Z"},
        tests=st(200) + [
            "pm.test('pause_minuten == 30', function () { pm.expect(pm.response.json().pause_minuten).to.eql(30); });",
            "pm.test('dauer_minuten == 331', function () { pm.expect(pm.response.json().dauer_minuten).to.eql(331); });",
        ]),
    start_A("2026-06-20T10:00:00Z", "(setup) Buchung A starten (10:00) fuer Zukunfts-End-Test"),
    req("TC-STOP-07 [DEF-01] Stop mit end_zeit in der Zukunft (2030) -> ERWARTET 400", "POST", ["zeit", "stop"], auth="A",
        b={"end_zeit": "2030-01-01T00:00:00Z"},
        tests=["pm.test('DEF-01: end_zeit in Zukunft muss 400 ergeben (analog start_zeit)', function () { pm.response.to.have.status(400); });"]),
])

f_patch = folder("3 — Korrektur (PATCH /zeit/{id})", [
    req("TC-PATCH-01 Eigene beendete Buchung: Notiz aendern -> 200", "PATCH", ["zeit", "{{zid360}}"], auth="A",
        b={"notiz": "korrigiert"},
        tests=st(200) + ["pm.test('notiz uebernommen', function () { pm.expect(pm.response.json().notiz).to.eql('korrigiert'); });"]),
    req("TC-PATCH-02 pause == Spanne (360) -> 200, Dauer 0", "PATCH", ["zeit", "{{zid360}}"], auth="A",
        b={"pause_minuten": 360},
        tests=st(200) + [
            "pm.test('pause_minuten == 360', function () { pm.expect(pm.response.json().pause_minuten).to.eql(360); });",
            "pm.test('dauer_minuten == 0', function () { pm.expect(pm.response.json().dauer_minuten).to.eql(0); });",
        ]),
    req("TC-PATCH-03 pause == Spanne+1 (361) -> 400", "PATCH", ["zeit", "{{zid360}}"], auth="A",
        b={"pause_minuten": 361}, tests=st(400) + errcode("bad_request")),
    req("TC-PATCH-04 pause_minuten = -1 -> 400 (Validierung min 0)", "PATCH", ["zeit", "{{zid360}}"], auth="A",
        b={"pause_minuten": -1}, tests=st(400) + errcode("validation_error")),
    # Ownership: B-Buchung anlegen + ID holen
    req("(setup) Worker B: Buchung starten", "POST", ["zeit", "start"], auth="B", b={},
        tests=["pm.test('start ok/bereits laufend', function () { pm.expect(pm.response.code).to.be.oneOf([201,409]); });"]),
    req("(setup) Worker B: laufende Buchung-ID holen", "GET", ["zeit", "laufend"], auth="B",
        tests=["pm.test('200', function () { pm.response.to.have.status(200); });"]
              + setvar("zidB", "pm.response.json() ? pm.response.json().id : ''")),
    req("TC-PATCH-05 Fremde Buchung (Worker B) patchen -> 404", "PATCH", ["zeit", "{{zidB}}"], auth="A",
        b={"notiz": "hack"}, tests=st(404) + errcode("not_found")),
    req("TC-PATCH-06 Unbekannte {id} -> 404", "PATCH", ["zeit", "019ee601-0000-7000-8000-000000000000"], auth="A",
        b={"notiz": "x"}, tests=st(404) + errcode("not_found")),
    req("TC-PATCH-07 Maliforme {id} -> 400", "PATCH", ["zeit", "not-a-uuid"], auth="A",
        b={"notiz": "x"}, tests=st(400) + errcode("bad_request")),
    req("TC-PATCH-08 start_zeit nach bestehendem end_zeit -> 400 (korrekt behandelt)", "PATCH", ["zeit", "{{zid360}}"], auth="A",
        b={"start_zeit": "2026-06-20T15:00:00Z"}, tests=st(400) + errcode("bad_request")),
    # DEF-02: Pause auf laufender Buchung
    req("(setup) Buchung A starten (laufend)", "POST", ["zeit", "start"], auth="A", b={},
        tests=["pm.test('start ok/bereits laufend', function () { pm.expect(pm.response.code).to.be.oneOf([201,409]); });"]),
    req("(setup) laufende Buchung-ID A holen", "GET", ["zeit", "laufend"], auth="A",
        tests=["pm.test('200', function () { pm.response.to.have.status(200); });"]
              + setvar("zidRun", "pm.response.json() ? pm.response.json().id : ''")),
    req("TC-PATCH-09 [DEF-02] pause_minuten auf laufender Buchung -> ERWARTET 400", "PATCH", ["zeit", "{{zidRun}}"], auth="A",
        b={"pause_minuten": 30},
        tests=["pm.test('DEF-02: Pause auf laufender Buchung darf nicht akzeptiert werden (400)', function () { pm.response.to.have.status(400); });"]),
    reset_A(),
])

f_list = folder("4 — Laufend & Liste (GET)", [
    start_A("2026-06-20T13:00:00Z", "(setup) Buchung A starten (13:00) fuer /laufend"),
    req("TC-LIST-01 /laufend bei laufender Buchung -> 200 mit Objekt", "GET", ["zeit", "laufend"], auth="A",
        tests=st(200) + [
            "pm.test('id vorhanden', function () { pm.expect(pm.response.json().id).to.be.a('string'); });",
            "pm.test('end_zeit null', function () { pm.expect(pm.response.json().end_zeit).to.eql(null); });",
        ]),
    reset_A(),
    req("TC-LIST-02 /laufend ohne laufende Buchung -> 200, leer/null", "GET", ["zeit", "laufend"], auth="A",
        tests=st(200) + ["pm.test('Body leer oder null', function () { var t = pm.response.text().trim(); pm.expect(t === '' || t === 'null').to.be.true; });"]),
    req("TC-LIST-03 Eigene Liste mit gueltigem Zeitfilter -> 200 Array", "GET", ["zeit", ""], auth="A",
        query=[("von", "2026-06-01T00:00:00Z"), ("bis", "2026-07-01T00:00:00Z")],
        tests=st(200) + ["pm.test('Array', function () { pm.expect(Array.isArray(pm.response.json())).to.be.true; });"]),
    req("TC-LIST-04 Eigene Liste mit ungueltigem von -> 400", "GET", ["zeit", ""], auth="A",
        query=[("von", "NOPE")], tests=st(400) + errcode("bad_request")),
])

f_auth = folder("5 — Auth & Rolle", [
    req("TC-AUTH-01 /zeit/start ohne Token -> 401", "POST", ["zeit", "start"], auth=None,
        b={}, tests=st(401) + errcode("unauthorized")),
    req("TC-AUTH-02 /admin/zeit mit Worker-Token -> 403", "GET", ["admin", "zeit"], auth="A",
        tests=st(403) + errcode("forbidden")),
    req("TC-AUTH-03 /admin/zeit mit Admin-Token (Gegenprobe) -> 200", "GET", ["admin", "zeit"], auth="admin",
        tests=st(200)),
])

collection = {
    "info": {
        "name": "Prekaj-Zeiterfassung — A4 Zeiterfassung start/stop",
        "description": "Spezifikationsbasierte Black-Box-Testfaelle (Aufgabe 4) fuer den Anwendungsfall 'Zeiterfassung starten/beenden'. Setup legt Testdaten an; DEF-01/DEF-02 schlagen bewusst fehl (gefundene Abweichungen).",
        "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
    },
    "event": [{"listen": "prerequest", "script": {"type": "text/javascript", "exec": COLLECTION_PREREQ}}],
    "variable": [{"key": k, "value": ""} for k in COLLECTION_VARS],
    "item": [setup, f_start, f_stop, f_patch, f_list, f_auth],
}

with open("zeiterfassung.postman_collection.json", "w", encoding="utf-8") as fh:
    json.dump(collection, fh, ensure_ascii=False, indent=2)
    fh.write("\n")

n_tc = sum(1 for fol in collection["item"] for it in fol["item"] if it["name"].startswith("TC-"))
print("OK: zeiterfassung.postman_collection.json geschrieben — %d Testfälle (TC-*)." % n_tc)
