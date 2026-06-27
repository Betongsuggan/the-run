# DSR-process: hantering av rättighetsbegäranden

Den här processen beskriver hur Ingmarsöloppet hanterar GDPR-rättigheter — registerutdrag, rättelse, radering, begränsning, dataportabilitet, invändning. De flesta begäranden går automatiskt via `/my-data` (självbetjäning); resten kräver manuell hantering enligt nedan.

## 1. SLA

GDPR Art. 12(3) sätter ramarna:

| Steg | Tid | Anmärkning |
|---|---|---|
| Mottagningsbekräftelse | **5 arbetsdagar** | Internt SLA — tightare än lagkravet. |
| Fullgjord begäran | **30 dagar** från mottagning | Lagkrav (kalenderdagar, inte arbetsdagar). |
| Komplexa fall | upp till **90 dagar** | Förlängning kräver att vi meddelar den registrerade inom 30 dagar med motivering. |

Räkningen börjar när begäran kommer in — inte när vi börjar arbeta med den.

## 2. Automatiserade självbetjäningskanaler (`/my-data`)

De flesta rättigheterna är redan automatiserade. Den registrerade behöver INTE höra av sig till oss för:

| Rättighet | Hur i `/my-data` |
|---|---|
| Rätt till tillgång (Art. 15) | Logga in via magiclänk → "Min data" visar allt vi har. |
| Rätt till rättelse (Art. 16) | Logga in → redigera namn, kön, födelsedatum direkt. |
| Rätt till radering (Art. 17) | Logga in → "Radera mitt konto" eller "Radera den här löparen". 30 dagars ångerfönster. |
| Rätt till dataportabilitet (Art. 20) | Logga in → "Exportera mina uppgifter (JSON)". |
| Återkalla samtycke (Art. 7.3) | Logga in → kryssruta för nyhetsbrev / publika resultat. |

**Identitetsverifiering** sker via magiclänk skickad till den mejladress som finns på kontot. Den som har tillgång till inkorgen anses tillräckligt verifierad — samma princip som standardiserade lösenordsåterställningar i de flesta tjänster.

## 3. Manuell hantering — när krävs det?

Begäran kräver manuell hantering när:

- **Personen har ingen åtkomst till sin gamla e-postadress.** T.ex. en gammal jobb-adress som inte längre fungerar. Magiclänken kan inte levereras.
- **Begäran kräver "begränsning av behandling" (Art. 18).** Vi har ingen GUI för det än — det är ovanligt och hanteras manuellt.
- **Invändning mot publika resultat eller nyhetsbrev sker via e-post** snarare än `/my-data` (om personen inte vill logga in).
- **Begäran kommer från någon annan än den registrerade** (t.ex. arvingar, advokater, vårdnadshavare). Extra identitetskontroll krävs.
- **Begäran är oklar eller kombinerad** ("ge mig allt och radera lite") — ofta enklare att svara via mejl och styra in personen i självbetjäningen.

## 4. Mottagandeflöde

```
[ ] Begäran inkommer (mejl, brev, eller via webbformulär)
[ ] Logga begäran i loggboken (se § 6 nedan)
[ ] Identitetsverifiering (se § 5 nedan)
[ ] Bekräftelsemejl inom 5 arbetsdagar (mall A nedan)
[ ] Utför begäran (eller meddela förlängning vid komplexitet)
[ ] Slutsvar inom 30 (eller 90) dagar (mall B-F nedan)
[ ] Markera begäran avslutad i loggboken
```

## 5. Identitetsverifiering

GDPR Art. 12(6): "skäliga åtgärder" för att verifiera identiteten. Inte ett bevis utöver rimligt tvivel — bara skälig säkerhet.

**Vår standard:**
- **Primär metod:** den registrerade triggar magiclänk på `/my-data` med sin gamla e-post. Klick → identitet verifierad.
- **Sekundär metod (om gamla e-posten inte fungerar):** be om en kompletterande uppgift som bara den registrerade rimligen kan veta — t.ex. tidigare loppdatum + ungefärligt startnummer, eller tidpunkt för senaste anmälan. Korsvalidera mot vår databas.
- **Avvik aldrig från principen "minst tillräcklig kontroll"**: be inte om passkopia eller personnummer om det inte krävs. Det är att samla in mer PII, inte mindre.

**Misstänkt impersonation:** om något känns fel (begäran från en helt ny e-post som säger "jag är X" utan att kunna verifiera något) — neka begäran med motivering. Be personen genomföra DSR-flödet från en e-post de har bevisad kontroll över. Logga avslag som "ej verifierad" i loggboken.

## 6. Loggbok

Alla manuella DSR-begäranden loggas i `docs/incidents/dsr-log-<år>.md` (eller en motsvarande struktur). Loggen är intern; den registrerade ser inte den. Schema:

```
| Datum mottagen | Begärt av (e-post/namn) | Typ | Verifierad? | Datum bekräftad | Datum fullgjord | Anteckningar |
|---|---|---|---|---|---|---|
| 2026-08-15 | birger@example.com | Radering | Ja (magiclänk) | 2026-08-16 | 2026-08-16 | Kontot raderat via /my-data; ingen manuell åtgärd. |
| 2026-09-02 | alice@example.com | Tillgång | Nej | — | — | Avslag — kunde ej verifiera. Bett kontakt via gammal e-post. |
```

Audit-tabellen i DynamoDB (`the-run-audit`) loggar automatiskt självbetjäningshändelser; den här filen kompletterar med manuella interaktioner.

## 7. Svarsmallar

### Mall A — mottagningsbekräftelse (inom 5 arbetsdagar)

```
Ämne: Vi har tagit emot din begäran — Ingmarsöloppet

Hej <namn>,

Tack för din begäran. Vi har registrerat den 2026-MM-DD och kommer
att svara senast 2026-MM-DD (30 dagar enligt GDPR).

För det snabbaste flödet kan du själv hantera de flesta rättigheter
direkt på https://ingmarsoloppet.se/my-data — vi mejlar en
inloggningslänk till den adress du har på kontot.

Om din begäran kräver manuell hantering ber vi om följande för att
verifiera din identitet:
<lista vad som efterfrågas, om något>

Med vänliga hälsningar,
Ingmarsöloppet
kontakt@ingmarsoloppet.se
```

### Mall B — registerutdrag (Art. 15)

```
Ämne: Ditt registerutdrag — Ingmarsöloppet

Hej <namn>,

Här är allt vi har om dig:

Konto:
- E-post: <e-post>
- Konto skapat: <datum>
- Språkinställning: <sv/en>
- Marknadssamtycke: <ja/nej> (senast ändrat <datum>)

Löpare under ditt konto:
<för varje löpare: namn, födelsedatum, kön, publika-resultat-samtycke>

Anmälningar:
<lista per lopp + sluttid om finished>

Aktivitetslogg (senaste 24 mån):
<utdrag>

Du kan när som helst själv exportera detta som JSON via /my-data.

Du har också rätt till rättelse, radering, dataportabilitet,
begränsning och invändning. Mer info: https://ingmarsoloppet.se/privacy

Med vänliga hälsningar,
Ingmarsöloppet
```

### Mall C — bekräftelse av radering

```
Ämne: Bekräftelse — uppgifterna är raderade

Hej <namn>,

Vi bekräftar att din begäran om radering är genomförd. Konkret:

- Konto: raderat (mjukraderat med 30 dagars ångerfönster fram till
  <datum>; därefter permanent borta).
- Löpare: <X> löpare raderade.
- Anmälningar utan tid: borttagna.
- Anmälningar med tid: anonymiserade — startnummer, tid och åldersklass
  finns kvar i resultatlistan utan koppling till dig.

Om du ångrar dig inom 30 dagar kan du återställa kontot via en
inloggningslänk till samma e-postadress. Efter det är data permanent
borta — det går inte att hämta tillbaka.

Du har rätt att klaga till Integritetsskyddsmyndigheten (IMY):
https://imy.se

Med vänliga hälsningar,
Ingmarsöloppet
```

### Mall D — invändning / återkallat samtycke

```
Ämne: Bekräftelse — inställningar uppdaterade

Hej <namn>,

Vi har uppdaterat dina samtycken:
- Nyhetsbrev: <ja/nej>
- Publika resultat: <ja/nej per löpare>

Du kan när som helst ändra dessa själv via /my-data.

Med vänliga hälsningar,
Ingmarsöloppet
```

### Mall E — förlängning till 90 dagar

```
Ämne: Förlängning av handläggningstid — Ingmarsöloppet

Hej <namn>,

På grund av <kort motivering — komplexitet, behov av manuell
verifiering, antal löpare under kontot>, behöver vi förlänga
handläggningstiden för din begäran från 2026-MM-DD med upp till
60 dagar (totalt 90 dagar från mottagandet enligt GDPR Art. 12.3).

Nytt slutdatum: 2026-MM-DD.

Med vänliga hälsningar,
Ingmarsöloppet
```

### Mall F — avslag (otillräcklig identitetsverifiering)

```
Ämne: Vi kan inte verifiera din begäran — Ingmarsöloppet

Hej,

Vi har tagit emot din begäran men kan inte verifiera att du är den
person uppgifterna gäller. Av integritetsskäl kan vi inte lämna ut,
ändra eller radera personuppgifter utan tillräcklig säkerhet om
identiteten.

Det enklaste sättet att verifiera dig är att besöka
https://ingmarsoloppet.se/my-data och begära en inloggningslänk
till den e-postadress du har använt för anmälan.

Om du inte längre har tillgång till den adressen — svara på det här
mejlet med:
- Ditt namn och födelsedatum.
- Ungefärligt datum för senaste anmälan eller deltagande.
- Ungefärligt startnummer om du minns det.

Med vänliga hälsningar,
Ingmarsöloppet
```

## 8. Klagorätt

Vi informerar i varje slutsvar om rätten att klaga till IMY: https://imy.se. Den registrerade behöver INTE försöka lösa något med oss först — IMY tar emot klagomål direkt.

## 9. När en begäran INTE behöver fullgöras

GDPR Art. 12(5)(b) och Art. 17(3) listar undantag. De vanligaste för oss:

- **Manifestabel ogrundad eller överdriven begäran.** T.ex. samma person som begär utdrag varje vecka. Vi kan ta ut en rimlig avgift eller neka — i praktiken talar vi med personen först.
- **Vi har en kvarstående rättslig grund för att behålla data.** T.ex. ofullständiga räkenskaper (för bokföringslagen, 7 år). Behåller vi data efter en raderingsbegäran måste vi förklara varför.

**Vi har ingen sådan kvarstående grund för loppdata** — alla anmälningar är rena privata avtal, ingen bokföringspliktig handling. En radering är alltid genomförbar.

## 10. Granskning

Den här processen granskas:

- Årligen, i samband med ROPA-granskning.
- När en begäran ger oss känslan av att processen brister (lägg till en åtgärdspunkt i loggboken).

---

**Senast granskad:** 2026-06-26.
