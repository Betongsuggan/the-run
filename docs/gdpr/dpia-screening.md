# DPIA-screening — Ingmarsöloppet

> Partially auto-generated. The data-flow + categories sections come
> from struct tags in `backend/internal/models/` (re-run `just gen-gdpr`
> to refresh). Risk-analysis prose between `<!-- gdpr:prose:NAME -->`
> markers is preserved across regenerations — that's where the operator
> writes the judgement calls.

## 1. Översikt
<!-- gdpr:prose:overview start -->

Det här dokumentet är en GDPR Art. 35-screening — en första bedömning av om Ingmarsöloppets behandling av personuppgifter kräver en full konsekvensbedömning (DPIA). Tjänsten är ett litet, årligt motionslopp (<500 deltagare/år) som drivs av en enskild arrangör. Behandlingen som beskrivs nedan omfattar namn, födelsedatum, kön, e-post och resultatdata för deltagare samt målsmans e-post och en aktiv samtyckesklick för deltagare under 13 år.

Inget tredjepartsverktyg för spårning, ingen profilering, inga särskilda kategorier av personuppgifter (Art. 9), ingen automatiserad beslutsfattning (Art. 22). All behandling sker i AWS region eu-north-1 (Stockholm); ingen överföring till tredje land. Den enda underbiträden är AWS EMEA SARL.

<!-- gdpr:prose:overview end -->

## 2. Vad behandlas? (auto-genererad inventering)

Följande personuppgiftskategorier behandlas, härledda från `gdpr:`-taggar på modellfält:

- **Kontaktuppgifter:** `Account.Email`, `AdminInvitation.Email`, `AdminInvitation.InvitedByMail`
- **Identifierande uppgifter:** `Registration.RunnerID`, `Runner.BirthDate`, `Runner.Gender`, `Runner.Name`
- **Inloggningsuppgifter:** `Account.MFASecret`, `Account.PasswordHash`, `MagicToken.ID`
- **Beteendedata:** `Registration.Bib`, `Registration.FinishSeconds`, `Registration.Splits`
- **Driftsdata:** `MagicToken.ContextID`
- **Loggdata:** `AuditRow.Action`, `AuditRow.Actor`, `AuditRow.Diff`, `AuditRow.Summary`, `AuditRow.TargetID`, `AuditRow.TargetType`

## 3. För vilka ändamål? (auto-genererad)

- **Anmälan och löpadministration** — Anmälan och löpadministration — startlista, åldersklass, kommunikation om loppet och resultat. _(rättslig grund: Avtal (Art. 6.1.b))_
- **Publika resultatlistor** — Publika resultatlistor — visning av namn, åldersklass och tid på webben (opt-out). _(rättslig grund: Berättigat intresse (Art. 6.1.f))_
- **Transaktionsmejl** — Transaktionsmejl — anmälningsbekräftelser, målsmanslänkar, lösenordsåterställning. _(rättslig grund: Avtal (Art. 6.1.b))_
- **Nyhetsbrev** — Nyhetsbrev — information om kommande lopp och relaterade nyheter. _(rättslig grund: Samtycke (Art. 6.1.a))_
- **Hantering av rättighetsbegäranden** — Hantering av rättighetsbegäranden — magiclänk-inloggning på /my-data, export, rättelse, radering. _(rättslig grund: Berättigat intresse (Art. 6.1.f))_
- **Administratörsautentisering** — Administratörsautentisering — lösenord och TOTP-MFA för adminpanelen. _(rättslig grund: Avtal (Art. 6.1.b))_
- **Bot- och spamskydd** — Bot- och spamskydd — kortvarig IP-räkning vid anmälningsformuläret och inloggningsförsöksspårning. _(rättslig grund: Berättigat intresse (Art. 6.1.f))_
- **Aktivitetslogg** — Aktivitetslogg — admin- och DSR-händelser för spårbarhet och utredning av incidenter. _(rättslig grund: Berättigat intresse (Art. 6.1.f))_

## 4. Berörda registrerade och särskilda kategorier
<!-- gdpr:prose:subjects start -->

**Tre kategorier av registrerade:**

- **Löpare (huvudgrupp)** — vuxna och minderåriga deltagare. Volym: storleksordningen 500/år. Nya löpare per säsong dominerar; en mindre andel är återkommande deltagare med flerårig historik.
- **Vårdnadshavare** — målsmän som anmäler deltagare under 13 år. Endast e-postadress lagras (på samma `Account`-post som löparen kopplas till). Vi mejlar en magiclänk för aktivt samtycke; ingen anmälan blir aktiv utan klick. Vårdnadshavarens kontaktuppgifter är funktionellt begränsade till denna roll.
- **Administratörer** — arrangören och eventuella biträden. Lagras med bcrypt-hashat lösenord och obligatorisk TOTP-MFA. Volym: enstaka konton.

**Sårbar grupp:** deltagare under 13 år. Hanteras via målsmans-samtyckesflödet (Art. 8) och redaktion av namn i publika resultat (endast förnamn + initial visas, oavsett samtycke).

**Inga särskilda kategorier av personuppgifter** (Art. 9) behandlas. Hälsodata kring konditionsförmåga härleds aldrig — sluttider är prestationsdata kopplade till lopp, inte hälsodata om individen.

<!-- gdpr:prose:subjects end -->

## 5. Nödvändighet och proportionalitet
<!-- gdpr:prose:necessity start -->

Varje fält som samlas in är minimalt nödvändigt för att fullgöra anmälningsavtalet eller skydda löparens rättigheter:

- **Namn + födelsedatum + kön** — krävs för åldersklass, identifiering på startlistan, och resultatpublicering. Datauppgifter som beräknats istället (t.ex. enbart åldersklass) skulle förlora återkoppling till individen, vilket gör DSR omöjligt.
- **E-post** — DSR-ankarpunkt (Art. 15, 16, 17) och anmälningsbekräftelse. Utan e-post skulle vi inte kunna verifiera identitet vid begäranden.
- **IP-adress** — kortlivad räknare för bot-skydd. Inte kopplad till identitet; rens via TTL inom 15 minuter.
- **Lösenord + TOTP** — endast för administratörskonton. Inte för löpare. Behovsanpassad till tjänstens drift, inte till deltagandet.

Proportionalitet är hög: ingen profilering, ingen återpaketering av data, inga delningar med annonseringsplattformar. Lagringen är begränsad genom en rullande 36-månadersmodell (anonymisering vid utebliven återanmälan) och DSR-driven radering.

<!-- gdpr:prose:necessity end -->

## 6. Identifierade risker
<!-- gdpr:prose:risks start -->

Identifierade risker, rangordnade efter sannolikhet × konsekvens:

1. **Återidentifiering via offentliga resultatlistor (medel/låg).** Resultatlistor visar fullständigt namn, åldersklass och tid som standard. En löpare som vill stå anonym men missar opt-out kan dyka upp i sökmotorer. Konsekvens: relativt begränsad — inga känsliga uppgifter — men oönskad. Mitigeras genom opt-out på `/my-data` + förvalda barn-redaktioner.
2. **Förlust av kontaktdata vid kontoövertagande (låg).** Adminkonto med läckt lösenord kan se alla löpares e-post. Konsekvens: spam-vektor mot deltagare. Mitigeras genom bcrypt + obligatorisk TOTP + audit-logg.
3. **Felaktig publicering av minderåriga (låg).** En bug i redaktionslogiken (`backend/internal/api/redact.go`) skulle kunna visa fullständigt namn på <13-deltagare. Konsekvens: brott mot barnskydd. Mitigeras genom befintliga enhetstester (`redact_test.go`) och explicit "förnamn + initial"-regel oavsett samtycke.
4. **Bot-driven skräppost-flöde (låg).** Anmälningsendpointet kan översvämmas, vilket producerar fake-PII som vi sedan måste rensa. Mitigeras med honeypot + per-IP-throttling + API Gateway burst limit.
5. **Glömd anonymisering vid utebliven återanmälan (låg).** Retention-sweep-Lambdan kan misslyckas tyst om DynamoDB returnerar ett fel. Konsekvens: PII finns kvar längre än policy. Mitigeras med dagligt schemalagt körning + CloudWatch-larm + manuell granskning kvartalsvis.
6. **Magiclänk-läcka (mycket låg).** En vidarebefordrad DSR- eller målsmanslänk ger åtkomst till motsvarande funktion. Mitigeras med en-shot-användning + 30-minuters TTL för DSR / 7-dagars för målsmans-samtycke + invalidation efter användning.

<!-- gdpr:prose:risks end -->

## 7. Åtgärder för att minska risker

Tekniska åtgärder (auto-genererade från `internal/gdpr/storage.go`):

- Kryptering i vila — AWS-hanterad nyckel på samtliga DynamoDB-tabeller (SSE).
- Kryptering under överföring — HTTPS via CloudFront och API Gateway.
- Säkerhetskopior — 35-dagars Point-in-Time Recovery på alla PII-bärande tabeller.
- Adminautentisering — bcrypt (kostnad 12) plus obligatorisk TOTP-MFA.
- Brute-force-skydd — 5 misslyckade försök ger 15 minuters låsning per konto.
- Bot- och spamskydd — honeypot-fält och per-IP-throttling på registrering.
- Aktivitetsloggar — 24 månaders retention på admin- och DSR-händelser.
- PII utelämnas från applikationsloggar — endast UUID-identifierare loggas.
- Loggretention — 30 dagar i CloudWatch Logs.

Organisatoriska och kompletterande åtgärder:
<!-- gdpr:prose:mitigations start -->

- **Inbyggt dataskydd (Art. 25).** Datamodellen separerar `Account` (kontaktidentitet) från `Runner` (deltagare) så att en vårdnadshavare kan hantera flera barn under ett konto utan att barnen får egna login-anchored konton.
- **Versionerad sekretesspolicy.** Varje samtycke pekar på en specifik policy-revision (`policyId`, `policyRevision`). När admin gör en ändring bevaras den exakta texten löparen accepterade i en separat snapshot-tabell.
- **Audit-logg på admin- och DSR-händelser.** 24-månaders retention. Synlig för användaren under "Min aktivitet" på `/my-data`.
- **Princip om dataminimering i loggar.** Applikationsloggar innehåller endast UUID-identifierare; namn, e-post och födelsedatum loggas aldrig.
- **Säkerhetsrutiner.** Adminkonton skapas via terminal (`just create-admin`), inte via formulär. Lösenord lagras aldrig i Pulumi-state eller miljövariabler.
- **Granskning.** Generera och granska denna DPIA-screening tillsammans med ROPA en gång per år eller vid större kodförändringar (nya fält, nya behandlingsändamål).
- **Incidenthantering.** Vid misstanke om brott: följ runbook (kommande slice B1.2). 72-timmars klocka mot IMY börjar vid medvetenhet.

<!-- gdpr:prose:mitigations end -->

## 8. Slutsats — krävs full DPIA?
<!-- gdpr:prose:conclusion start -->

**Slutsats: Full DPIA krävs inte.**

Bedömning mot IMY:s vägledning om Art. 35.3 och WP29:s nio kriterier:

| Kriterium | Träff? | Kommentar |
|---|---|---|
| Utvärdering eller poängsättning (profilering) | Nej | Ingen profilering. |
| Automatiserat beslutsfattande med rättsliga effekter | Nej | Endast Art. 22-undantaget icke-tillämpligt. |
| Systematisk övervakning | Nej | Inga fysiska eller digitala spårfunktioner. |
| Känsliga uppgifter eller mycket personliga uppgifter | Nej | Inga särskilda kategorier. |
| Storskalig databehandling | Nej | <500 deltagare/år. |
| Matchning eller kombination av datamängder | Nej | Ingen kombination med externa källor. |
| Sårbara registrerade | **Ja** | Deltagare <13 år finns. Mitigeras med målsmans-flöde + redaktion. |
| Innovativ användning av teknisk lösning | Nej | Standard webapp + DynamoDB. |
| När bearbetningen hindrar registrerade från utövande av rättigheter | Nej | DSR-flöde på `/my-data` täcker Art. 15–22. |

Endast ett kriterium triggas (sårbar grupp). WP29 rekommenderar full DPIA vid två eller fler kriterier. Med ett enda kriterium, mitigerat genom etablerade kontroller, bedöms behandlingen som låg risk och faller utanför kravet.

Granskning rekommenderas:

- Årligen, i samband med ROPA-granskning.
- Vid förändring som introducerar nya kategorier av personuppgifter eller nya behandlingar (drift detekteras automatiskt av `TestNoGdprDocDrift`).
- Vid förändring av tjänstens omfattning som närmar sig "storskalig" (t.ex. >5 000 deltagare/år).

_Datum för senaste manuella granskning: 2026-06-26 (initial bedömning)._

<!-- gdpr:prose:conclusion end -->
