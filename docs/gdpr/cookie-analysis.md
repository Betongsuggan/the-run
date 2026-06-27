# Cookie- och lokal-lagrings-analys

Den här filen dokumenterar Ingmarsöloppets analys av varför webbplatsen INTE behöver något cookie-samtyckesbanner. Den är en intern artefakt avsedd att försvara den slutsatsen vid en eventuell IMY-granskning.

## Rättslig grund

**EU:s ePrivacy-direktiv Art. 5(3)**, transponerad till svensk rätt som **6 kap. 18 § LEK** (Lag om elektronisk kommunikation):

> Uppgifter får lagras i eller hämtas från en användares terminalutrustning endast om användaren får tillgång till information om ändamålet med behandlingen och samtycker till den.
>
> Detta gäller inte sådan lagring eller åtkomst som **endast** syftar till att utföra eller underlätta överföringen av ett elektroniskt meddelande, eller **som är nödvändig för att tillhandahålla en tjänst som användaren uttryckligen har begärt**.

Två undantag från samtyckeskravet:
1. Strikt nödvändigt för att överföra meddelandet (t.ex. CDN routing-cookies).
2. Strikt nödvändigt för en tjänst användaren uttryckligen begärt (t.ex. inloggning).

Vägledning från **EDPB Guidelines 2/2023 om Art. 5(3)** och **IMY:s kakvägledning** preciserar tillämpningen.

## Inventering

### Server-satta kakor

| Namn | Sätts när | TTL | Flaggor | Ändamål |
|---|---|---|---|---|
| `the-run-session` | Efter administratörens lyckade lösenord + TOTP-inloggning | 8 h | HttpOnly, Secure, SameSite=Strict | Bär JWT för administratörssession. Utan den måste varje sidvisning be om nya autentiseringsuppgifter. |
| `the-run-dsr-session` | Efter att användaren klickat på magiclänken i sin e-post för `/my-data` | 2 h | HttpOnly, Secure, SameSite=Strict | Bär JWT för DSR-session. Utan den måste varje åtgärd på `/my-data` kräva en ny magiclänk. |
| `the-run-auth-challenge` | Mellan administratörens lösenordssteg och TOTP-steg | 5 min | HttpOnly, Secure, SameSite=Strict | Bär den korta utmaningen mellan de två stegen i inloggningen. Utan den kan flerstegsflödet inte fungera. |

**Källkod:** `backend/internal/auth/{session.go,challenge.go,dsr_session.go}` + `backend/internal/api/auth.go:buildCookie`.

`Secure`-flaggan sätts i produktion (`SecureCookies=true`); slås av endast lokalt när `INSECURE_COOKIES=1` exporteras för LocalStack-utveckling.

### Klient-satta värden i `localStorage`

| Nyckel | Sätts när | Innehåll | Ändamål |
|---|---|---|---|
| `the-run.locale` | Användaren klickar på språkväljaren i toppmenyn | `"sv"` eller `"en"` | Webbplatsen kommer ihåg språkvalet mellan besök. |
| `the-run.theme` | Användaren klickar på temaväljaren | `"light"`, `"dark"`, eller `"system"` | Webbplatsen kommer ihåg temavalet mellan besök. |

**Källkod:** `frontend/src/lib/i18n/state.svelte.ts` + `frontend/src/lib/theme/state.svelte.ts`.

**Inga andra `localStorage`-nycklar.** Verifierat med en grep mot `frontend/src/`. Inga `sessionStorage`-skrivningar någonstans. Inga `document.cookie`-skrivningar från klientsidan.

### Tredjeparter

Verifierat avsaknad (grep mot `frontend/src/` + `frontend/package.json`):

- Ingen Google Analytics (`gtag`, `analytics.js`).
- Ingen Facebook Pixel (`fbq`).
- Ingen Hotjar / Mixpanel / Segment / PostHog / Plausible / Fathom.
- Ingen Sentry, ingen errortrackerssdk.
- Inga reklamnätverk, inga sociala-medie-widgets, inga embedded YouTube/Vimeo-spelare.

Den enda tredjeparts-domänen sidan kommunicerar med vid drift är AWS (CloudFront för statiska tillgångar, API Gateway för anrop) — vilket är underbiträde, inte spårare. Se `docs/gdpr/subprocessors.md`.

## Bedömning per lagringspunkt

### Sessionskakorna (3 st)

Alla tre sessionskakor är **strikt nödvändiga för en tjänst som användaren uttryckligen begärt** — administratörsinloggning eller `/my-data`-inloggning. EDPB Guidelines 2/2023, recital 22: "session cookies that are strictly necessary for the requested service" omfattas av undantaget. **Inget samtycke krävs.**

### Funktionella preferenser i `localStorage` (2 st)

Detta är den grånare zonen. EDPB Guidelines 2/2023 § 3.2.2 listar **kriterier för när preferens-kakor kan vara undantagna**:

- Sätts endast efter att användaren aktivt gjort valet — ✓ (kakan finns inte före användaren klickar på språk/tema-väljaren).
- Begränsad till det enda syftet — ✓ (innehåller bara `"sv"` / `"light"` etc.).
- Delas inte med tredjepart — ✓ (rent klientsidigt, läses bara av vår egen webbplats).
- Innehåller inga identifierare som kan användas för spårning — ✓ (värdet är enumererat; ingen UUID, ingen tidsstämpel, ingen sessionsnyckel).

IMY:s ståndpunkt i deras allmänna kakvägledning är konsekvent med EDPB:s — funktionella preferenser som triggas av användarens aktiva val är undantagna. **Inget samtycke krävs.**

## Slutsats

**Inget cookie-samtyckesbanner krävs på Ingmarsöloppets webbplats.**

Alla server-satta kakor är strikt nödvändiga sessionskakor. De två `localStorage`-värdena är funktionella preferenser som triggas av användarens aktiva val och innehåller inga identifierare.

Däremot kvarstår **transparens-skyldigheten** enligt Art. 5(3) sista meningen och GDPR Art. 13 — vi måste informera användaren om vad som lagras. Det görs i sekretesspolicyn på `/privacy`, sektion 8.

## Granskningstriggers

Den här bedömningen måste omprövas om vi:

- Lägger till spårningsverktyg (Analytics, A/B-test, error-tracking).
- Inför ett socialt-medie-inbäddat innehåll (YouTube-spelare, Twitter-feed, Facebook like-knapp).
- Lägger till en `localStorage`-nyckel som innehåller en identifierare (UUID, sessionsnyckel utöver autentisering).
- Inför ett reklamnätverk eller affiliateprogram.

Att lägga till något ur listan ovan innebär INTE automatiskt att vi behöver ett banner — det innebär att den här analysen behöver göras om. Slutsatsen kan bli ett mer informativt notis-element eller ett fullt samtyckesgränssnitt beroende på vad som tillkommer.

## Verifiering

För att verifiera att den här analysen fortfarande stämmer:

```bash
# Inga spårningsverktyg i frontend-bundlen:
grep -rn "gtag\|analytics\|fbq\|hotjar\|mixpanel\|segment\|posthog\|sentry\|plausible\|fathom" frontend/src/ frontend/package.json

# Endast väntade localStorage-nycklar:
grep -rn "localStorage" frontend/src/ | grep -v "// "

# Cookies sätts bara via våra auth-helpers:
grep -rn "Set-Cookie\|SetCookie\|http.Cookie" backend/internal/
```

Alla tre kommandona ska producera exakt det som beskrivs i inventeringen ovan. En PR som ändrar resultatet av något av dem ska trigga en granskning av den här filen.

---

**Senast granskad:** 2026-06-26.
**Granskningscykel:** vid varje större frontend-eller-backend-ändring som rör tredjepartsintegrationer eller klientsidig lagring, och årligen vid ROPA-granskning.
