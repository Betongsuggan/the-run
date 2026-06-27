# Records of Processing Activities (ROPA)

> **Auto-generated.** To make changes, edit `gdpr:` struct tags
> in `backend/internal/models/` or the registries in
> `backend/internal/gdpr/`, then run `just gen-gdpr`. Do not edit
> this file by hand — the drift test in CI will reject any change
> that doesn't match what the generator would produce.

**Personuppgiftsansvarig:** Ingmarsöloppet  
**Kontakt:** kontakt@ingmarsoloppet.se  
**Tillsynsmyndighet:** Integritetsskyddsmyndigheten (IMY)

## Behandlingar

### 1. Anmälan och löpadministration (`registration`)

Anmälan och löpadministration — startlista, åldersklass, kommunikation om loppet och resultat.

- **Rättslig grund:** Avtal (Art. 6.1.b)
- **Kategorier av registrerade:** Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:**
  - Kontaktuppgifter: `Account.Email`
  - Identifierande uppgifter: `Registration.RunnerID`, `Runner.BirthDate`, `Runner.Gender`, `Runner.Name`
  - Inloggningsuppgifter: `MagicToken.ID`
  - Beteendedata: `Registration.Bib`
  - Driftsdata: `MagicToken.ContextID`
- **Lagringstid:** 36 månader rullande från senaste loppdeltagande. Klockan börjar om vid varje nytt lopp.

### 2. Publika resultatlistor (`public-results`)

Publika resultatlistor — visning av namn, åldersklass och tid på webben (opt-out).

- **Rättslig grund:** Berättigat intresse (Art. 6.1.f)
- **Kategorier av registrerade:** Löpare
- **Kategorier av personuppgifter:**
  - Identifierande uppgifter: `Registration.RunnerID`, `Runner.Gender`, `Runner.Name`
  - Beteendedata: `Registration.Bib`, `Registration.FinishSeconds`, `Registration.Splits`
- **Lagringstid:** Tills vidare om publika resultat tillåts (löparhistorik).

### 3. Transaktionsmejl (`comms`)

Transaktionsmejl — anmälningsbekräftelser, målsmanslänkar, lösenordsåterställning.

- **Rättslig grund:** Avtal (Art. 6.1.b)
- **Kategorier av registrerade:** Administratör, Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:**
  - Kontaktuppgifter: `Account.Email`
- **Lagringstid:** 36 månader rullande från senaste aktivitet (inloggning, anmälan eller lopp).

### 4. Nyhetsbrev (`marketing`)

Nyhetsbrev — information om kommande lopp och relaterade nyheter.

- **Rättslig grund:** Samtycke (Art. 6.1.a)
- **Kategorier av registrerade:** Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:**
  - Kontaktuppgifter: `Account.Email`
- **Lagringstid:** Tills samtycket återkallas (avregistrering från nyhetsbrevet).

### 5. Hantering av rättighetsbegäranden (`dsr`)

Hantering av rättighetsbegäranden — magiclänk-inloggning på /my-data, export, rättelse, radering.

- **Rättslig grund:** Berättigat intresse (Art. 6.1.f)
- **Kategorier av registrerade:** Administratör, Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:**
  - Kontaktuppgifter: `Account.Email`
  - Inloggningsuppgifter: `MagicToken.ID`
  - Driftsdata: `MagicToken.ContextID`
- **Lagringstid:** Kort engångslänk (30 minuter – 7 dagar beroende på syfte).

### 6. Administratörsautentisering (`admin-auth`)

Administratörsautentisering — lösenord och TOTP-MFA för adminpanelen.

- **Rättslig grund:** Avtal (Art. 6.1.b)
- **Kategorier av registrerade:** Administratör
- **Kategorier av personuppgifter:**
  - Inloggningsuppgifter: `Account.MFASecret`, `Account.PasswordHash`
- **Lagringstid:** 8 timmar (absolut livslängd för adminsession).

### 7. Bot- och spamskydd (`fraud-prevention`)

Bot- och spamskydd — kortvarig IP-räkning vid anmälningsformuläret och inloggningsförsöksspårning.

- **Rättslig grund:** Berättigat intresse (Art. 6.1.f)
- **Kategorier av registrerade:** Administratör, Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:** (denna behandling konsumerar inga taggade fält direkt — den styr cookies, sessioner eller skyddsåtgärder)
- **Lagringstid:** 15 minuter (misslyckade inloggningsförsök).

### 8. Aktivitetslogg (`audit`)

Aktivitetslogg — admin- och DSR-händelser för spårbarhet och utredning av incidenter.

- **Rättslig grund:** Berättigat intresse (Art. 6.1.f)
- **Kategorier av registrerade:** Administratör, Löpare, Vårdnadshavare
- **Kategorier av personuppgifter:**
  - Loggdata: `AuditRow.Action`, `AuditRow.Actor`, `AuditRow.Diff`, `AuditRow.Summary`, `AuditRow.TargetID`, `AuditRow.TargetType`
- **Lagringstid:** 24 månader.

## Underbiträden

| Namn | Roll | Region | Tjänster | DPA |
|---|---|---|---|---|
| Amazon Web Services EMEA SARL | Drift, lagring och e-postutskick | eu-north-1 (Stockholm) + CloudFront-edge inom EES | DynamoDB, Lambda, API Gateway, CloudFront, SES, Secrets Manager, CloudWatch Logs | Accepterad via AWS Artifact |

## Tabeller och lagringsläge

| Tabell | SSE | PITR | TTL | Innehåll |
|---|---|---|---|---|
| `the-run-runners` | ✓ | ✓ | – | Löparuppgifter (namn, födelsedatum, kön, samtycke). |
| `the-run-registrations` | ✓ | ✓ | – | Anmälningar och resultat per löpare och klass. |
| `the-run-accounts` | ✓ | ✓ | – | Konton (e-post, lösenord, MFA, marknadssamtycke). |
| `the-run-events` | ✓ | – | – | Loppen — namn, år, datum, plats. |
| `the-run-races` | ✓ | – | – | Klasser inom varje lopp (5K, 10K, barnens lopp). |
| `the-run-auth-attempts` | ✓ | – | `expiresAt` | Misslyckade inloggningsförsök — auto-rensas via TTL. |
| `the-run-magic-tokens` | ✓ | – | `expiresAt` | Engångslänkar (DSR, målsmanssamtycke, e-poständring) — auto-rensas via TTL. |
| `the-run-audit` | ✓ | – | – | Aktivitetsloggar för admin- och DSR-händelser. |
| `the-run-rate-limit` | ✓ | – | `expiresAt` | Korta IP-räknare för bot-skydd vid anmälan — auto-rensas via TTL. |
| `the-run-policies` | ✓ | ✓ | – | Versionerade sekretesspolicyer. |
| `the-run-policy-revisions` | ✓ | – | – | Bevarade tidigare versioner av policyer (snapshots). |
| `the-run-email-templates` | ✓ | ✓ | – | Versionerade textinnehåll för transaktionsmejl (kvitton, magic links). |
| `the-run-email-template-revisions` | ✓ | – | – | Bevarade tidigare versioner av mejlmallar (snapshots). |
| `the-run-admin-invitations` | ✓ | – | `expiresAt` | Pågående administratörsinbjudningar (e-post + hashad engångstoken) — auto-rensas via TTL efter 7 dagar. |

## Tekniska och organisatoriska säkerhetsåtgärder

- Kryptering i vila — AWS-hanterad nyckel på samtliga DynamoDB-tabeller (SSE).
- Kryptering under överföring — HTTPS via CloudFront och API Gateway.
- Säkerhetskopior — 35-dagars Point-in-Time Recovery på alla PII-bärande tabeller.
- Adminautentisering — bcrypt (kostnad 12) plus obligatorisk TOTP-MFA.
- Brute-force-skydd — 5 misslyckade försök ger 15 minuters låsning per konto.
- Bot- och spamskydd — honeypot-fält och per-IP-throttling på registrering.
- Aktivitetsloggar — 24 månaders retention på admin- och DSR-händelser.
- PII utelämnas från applikationsloggar — endast UUID-identifierare loggas.
- Loggretention — 30 dagar i CloudWatch Logs.

## Överföringar till tredje land

Inga. All data lagras i AWS region eu-north-1 (Stockholm) och distribueras via CloudFront-edge inom EES.
