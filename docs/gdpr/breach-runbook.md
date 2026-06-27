# Runbook: Personuppgiftsincident

Operationell handlingsplan när Ingmarsöloppet upptäcker eller misstänker att personuppgifter kommit på avvägar, ändrats eller blivit otillgängliga.

**Hela klockan från medvetenhet räknas i timmar — inte arbetsdagar.** Inkidentens första timme är dyrast: dokumentation, kommunikation och åtgärder i fel ordning gör en hanterbar händelse till en utdragen kris.

## 1. Vad räknas som personuppgiftsincident?

GDPR Art. 4(12): "en säkerhetsincident som leder till oavsiktlig eller olaglig förstöring, förlust eller ändring eller till obehörigt röjande av eller obehörig åtkomst till de personuppgifter som överförts, lagrats eller på annat sätt behandlats".

Tre konkreta exempel som kan inträffa här:

1. **Obehörig åtkomst.** Adminlösenord läcker (t.ex. via phishing) och någon utomstående loggar in på adminpanelen. Hela registreringsdatabasen är åtkomlig för angriparen från det ögonblicket.
2. **Förlust av integritet.** Ett bugfix ändrar oavsiktligt PII på fel löpare (t.ex. en bulk-update PATCH som råkar matcha för många rader).
3. **Förlust av tillgänglighet.** DynamoDB-tabell raderas av misstag i en `pulumi destroy` mot fel stack. PITR återställer 35 dagar tillbaka, men data nyare än senaste PITR-snapshot är borta för alltid.

**Inte en incident:**
- En bot fyller anmälningsformuläret med skräpdata → honeypot+throttling stoppar.
- En användare själv begär radering via `/my-data` → DSR-flödet är avsett.
- CloudWatch-larm för CPU/latens utan koppling till PII → driftsincident, inte personuppgiftsincident.

## 2. Detektionskällor

| Källa | Typisk händelse |
|---|---|
| CloudWatch Logs / Metrics | Plötslig spik i `admin.*` audit-händelser från en ny IP, eller 5xx-fel på `/registrations`. |
| Audit-tabellen (`the-run-audit`) | Mass-ändringar av runners/registrations utanför normalt mönster. |
| Användarrapport | Mejl från löpare som ser fel data om sig på `/my-data`. |
| Tredjeparts notifiering | AWS Trust & Safety, säkerhetsforskare, eller IMY. |
| Kodgranskning post-incident | En PR-granskning upptäcker att en tidigare deploy hade en bug som exponerat data. |

## 3. Triage-checklista (första 60 minuter)

```
[ ] Vem är ansvarig just nu? (skriv ner namnet i incident-loggen)
[ ] Vad är fakta? (ej spekulation — bara det vi vet)
[ ] Vad är okänt? (lista, för att undvika att bekräfta antaganden senare)
[ ] Är åtkomsten fortfarande öppen? Om ja → STÄNG NU.
[ ] Hur många registrerade är potentiellt drabbade? (>1, >10, >100?)
[ ] Vilka kategorier av personuppgifter? (kontakt? hälsa? barn?)
[ ] Tidpunkt för första exponering — tidigaste rimliga.
[ ] Tidpunkt vi blev medvetna — exakt, för 72h-klockan.
```

### Akuta åtgärder (parallellt)

- **Stäng åtkomstvägen.** Misstänkt komprometterat adminkonto: kör `just create-admin <email> '<ny-lösen>' force` — det roterar lösenordet och nollställer MFA-secret. Avsluta även eventuella aktiva sessions (kort sikt: rotera JWT-secreten i AWS Secrets Manager, vilket invaliderar alla cookies på världen).
- **Frys ändringar.** Inga `just deploy` förrän rotorsaken är förstådd. En andra deploy under utredning skapar otydlig spårbarhet.
- **Säkra bevisen.** Snapshot CloudWatch Logs för relevanta tidsperioden (`aws logs filter-log-events ...` → fil). Dumpa audit-tabellen (`the-run-audit`) för den dagen. Spara separat — inte i samma S3-bucket som tjänsten.

## 4. 72-timmarsklockan mot IMY

GDPR Art. 33(1): incidenten ska anmälas till IMY **utan oskäligt dröjsmål och senast 72 timmar efter att man fått kännedom om den**, om det är troligt att den medför en risk för fysiska personers rättigheter och friheter.

Klockan börjar gå **vid medvetenhet** — inte vid bekräftelse. Om en CloudWatch-larm pingar kl 13:00 och vi börjar utreda kl 14:00 efter lunch, är medvetenhetstidpunkten 13:00.

**Undantag från anmälningsplikten:** incidenten medför sannolikt INGEN risk för registrerades rättigheter (Art. 33(1) andra meningen). T.ex. om alla läckta uppgifter var pseudonymiserade/krypterade på ett sätt som gör dem obrukbara. Vår krypteringssituation (AWS-hanterad nyckel på vila) räcker INTE för det undantaget — angriparen kan ha åtkomst via samma API som har dekryptering.

### Anmälningsmall (svenska, för IMY)

Skicka via [imy.se/anmal](https://www.imy.se/verksamhet/dataskydd/anmal-personuppgiftsincident/) — den interaktiva formuläret kräver i princip följande:

```
Personuppgiftsansvarig: Ingmarsöloppet
Kontaktperson: <namn> · <e-post> · <telefon>

Tidpunkt för incidenten:
- Tidigaste rimliga exponering: <datum/tid>
- Vi fick kännedom: <datum/tid>
- Anmälan görs: <datum/tid>

Beskrivning av incidenten:
<3-5 meningar — vad hände, hur upptäcktes det, vilka åtgärder är vidtagna nu?>

Kategorier av personuppgifter som berörs:
<lista, t.ex.: namn, födelsedatum, kön, e-post>

Kategorier av registrerade:
<t.ex.: vuxna löpare, vårdnadshavare, deltagare under 13 år>

Ungefärligt antal berörda registrerade:
<antal eller intervall>

Ungefärligt antal berörda personuppgiftsposter:
<antal>

Trolig konsekvens:
<t.ex.: risk för riktade phishingförsök mot drabbade e-postadresser>

Vidtagna åtgärder (för att begränsa effekt):
<lista — lösenordsrotation, JWT-rotation, IP-blockering, etc.>

Föreslagna åtgärder (för att förhindra återupprepning):
<lista>

Har vi underrättat de registrerade?
<ja/nej + motivering>
```

## 5. När underrätta de registrerade direkt?

GDPR Art. 34: när incidenten **sannolikt leder till en hög risk** för registrerades rättigheter och friheter.

**Tröskelvärden vi tillämpar (intern policy):**
- Lösenord eller MFA-secrets i klartext exponerade → ALLTID underrätta.
- Namn + födelsedatum för deltagare under 13 år exponerade → ALLTID underrätta vårdnadshavare.
- E-postadresser och namn för fler än 50 vuxna löpare exponerade → underrätta.
- Endast aggregerade resultattider utan namn → underrätta INTE (låg risk).

### Underrättelsemall (svenska, till drabbade löpare)

```
Ämne: Information om en personuppgiftsincident hos Ingmarsöloppet

Hej <namn>,

Vi måste informera dig om att en säkerhetshändelse har påverkat dina personuppgifter
som vi har lagrade. Här är vad vi vet:

Vad hände:
<2-3 meningar, klarspråk>

Vilka uppgifter berördes (om just dig):
<lista>

Vad vi har gjort:
<lista — vad vi redan åtgärdat>

Vad du kan göra:
<konkreta råd — byt lösenord på återanvända konton, var uppmärksam på phishing, etc.>

Vi har anmält händelsen till Integritetsskyddsmyndigheten (IMY) i enlighet med GDPR.
Du har rätt att lämna in ett klagomål direkt till IMY: https://imy.se

Om du har frågor, kontakta oss på kontakt@ingmarsoloppet.se.

Med vänliga hälsningar,
Ingmarsöloppet
```

Skicka från `kontakt@ingmarsoloppet.se` via SES — INTE från ett externt e-postverktyg som behöver en separat databehandlare.

## 6. Post-incident (inom 1-2 veckor)

1. **Skriv post-mortem.** Tidslinje, rotorsak, vad gick rätt, vad gick fel, åtgärdspunkter. Lagra i `docs/incidents/<datum>-<kort-titel>.md`. Inkludera INTE PII från drabbade — referera bara IDs.
2. **Genomför åtgärder.** Varje åtgärdspunkt får en deadline; ingen punkt stängs utan en verifierbar förändring i kod, infra eller process.
3. **Uppdatera den här runbooken.** Saknades en checklista? Lägg till den. Var en regelfras otydlig? Skriv om den.
4. **Verifiera revisionsloggen.** Audit-tabellen ska kunna återrapportera HELA incidenten i efterhand. Saknade rader = brist i loggningen som ska åtgärdas.
5. **Granska behov av extern utredning.** Om incidenten involverar brott (t.ex. dataintrång enligt 4 kap. 9c § BrB) — överväg polisanmälan parallellt med IMY-anmälan.

## 7. Övning

Spela igenom den här runbooken en gång per år utan riktig incident: välj ett av exempelscenarierna ovan, sätt en kökstimer på 60 minuter, gå igenom checklistan med riktiga kommandon (utan att exekvera destruktiva). Notera friktionspunkter och åtgärda dem.

---

**Senast granskad:** 2026-06-26.
**Granskningscykel:** årlig, eller efter varje incident.
