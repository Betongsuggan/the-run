# Lagringsschema

> **Auto-generated** från `backend/internal/gdpr/retention.go` + `backend/internal/gdpr/storage.go`. Edit those files och kör `just gen-gdpr` för att uppdatera.

Det här schemat samlar alla lagringstider för personuppgifter i tjänsten. Varje rad pekar tillbaka på en Go-konstant som styr den faktiska beteendet — så schemat och koden kan inte drifta ifrån varandra.

## Lagringstider per kategori

| Nyckel | Beskrivning | Källa i kod |
|---|---|---|
| `account-inactivity-36mo` | 36 månader rullande från senaste aktivitet (inloggning, anmälan eller lopp). | 36 månader |
| `audit-24mo` | 24 månader. | 24 månader |
| `auth-attempt-15min` | 15 minuter (misslyckade inloggningsförsök). | 15m0s |
| `consent-until-withdrawn` | Tills samtycket återkallas (avregistrering från nyhetsbrevet). | tills handling (samtycke återkallat, konto raderat, etc.) |
| `dns-no-show-6mo` | 6 månader efter loppet (anmälningar utan start). | 6 månader |
| `magic-token-short` | Kort engångslänk (30 minuter – 7 dagar beroende på syfte). | tills handling (samtycke återkallat, konto raderat, etc.) |
| `runner-indefinite-if-consenting` | Tills vidare om publika resultat tillåts (löparhistorik). | tills handling (samtycke återkallat, konto raderat, etc.) |
| `runner-sliding-36mo` | 36 månader rullande från senaste loppdeltagande. Klockan börjar om vid varje nytt lopp. | 36 månader |
| `session-8h` | 8 timmar (absolut livslängd för adminsession). | 8h0m0s |

## Lagring per DynamoDB-tabell

Tabeller med automatisk TTL får raderna automatiskt borttagna av DynamoDB. Övriga tabeller städas av en daglig retention-Lambda (`backend/cmd/retention/`).

| Tabell | TTL-attribut | PITR (35d) | Hanteras av |
|---|---|---|---|
| `the-run-runners` | – | ✓ | retention-Lambda + DSR-flöde |
| `the-run-registrations` | – | ✓ | retention-Lambda + DSR-flöde |
| `the-run-accounts` | – | ✓ | retention-Lambda + DSR-flöde |
| `the-run-events` | – | – | retention-Lambda + DSR-flöde |
| `the-run-races` | – | – | retention-Lambda + DSR-flöde |
| `the-run-auth-attempts` | `expiresAt` | – | DynamoDB TTL (automatiskt) |
| `the-run-magic-tokens` | `expiresAt` | – | DynamoDB TTL (automatiskt) |
| `the-run-audit` | – | – | retention-Lambda + DSR-flöde |
| `the-run-rate-limit` | `expiresAt` | – | DynamoDB TTL (automatiskt) |
| `the-run-policies` | – | ✓ | retention-Lambda + DSR-flöde |
| `the-run-policy-revisions` | – | – | retention-Lambda + DSR-flöde |

## Lagringskedjan från löparens perspektiv

1. **Anmälan utan start (DNS/no-show):** raderas 6 månader efter loppdatumet.
2. **Genomfört lopp + tillåtelse till publika resultat:** sparas tills vidare som löparhistorik.
3. **Genomfört lopp utan tillåtelse till publika resultat:** anonymiseras 36 månader efter senaste loppet (rullande — klockan börjar om vid varje nytt lopp).
4. **Konto utan kvarvarande löpare:** raderas 30 dagar efter att sista löparen tagits bort (ångerfönster).
5. **Inaktivt konto:** mjuk-raderas efter 36 månader utan aktivitet (inloggning, anmälan, lopp). Användaren mejlas och har 30 dagar att återställa.
6. **Adminaktivitetslogg:** 24 månader.
7. **Säkerhetsloggar (CloudWatch):** 30 dagar.
8. **Säkerhetskopior (DynamoDB PITR):** 35 dagar på PII-bärande tabeller.

## Hur ändras schemat?

1. Uppdatera konstanten i `backend/internal/gdpr/retention.go`.
2. Kör `just gen-gdpr` — den här filen + ROPA uppdateras automatiskt.
3. Befintliga rader påverkas vid nästa retention-Lambda-körning (dagligen).
4. Notera ändringen i sekretesspolicyn (publicera en ny version via `/admin/policies` om förändringen är väsentlig).
