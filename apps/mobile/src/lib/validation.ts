export const MIN_AGE = 18;
export const MAX_AGE = 99;
export const PASSWORD_MIN = 8;
export const PASSWORD_MAX = 72;
export const BIO_MAX = 500;
export const FIRST_NAME_MAX = 40;
export const MAX_INTERESTS = 10;
export const MAX_PHOTOS = 6;
export const MESSAGE_MAX = 2000;
export const REPORT_DETAILS_MAX = 1000;
export const DISTANCE_MAX = 500;

export function normalizeEmail(raw: string): string {
  return raw.trim().toLowerCase();
}

export function isValidEmail(raw: string): boolean {
  const email = normalizeEmail(raw);
  return email.length <= 254 && /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email);
}

export function isValidPassword(password: string): boolean {
  return password.length >= PASSWORD_MIN && password.length <= PASSWORD_MAX;
}

export function isValidFirstName(raw: string): boolean {
  const name = raw.trim();
  return name.length >= 1 && name.length <= FIRST_NAME_MAX;
}

export type BirthDateError = 'incomplete' | 'invalid' | 'underage';

export type BirthDateResult =
  { ok: true; iso: string; age: number } | { ok: false; error: BirthDateError };

const pad = (n: number, width = 2): string => String(n).padStart(width, '0');

/** Age in whole years at `now` for a YYYY-MM-DD birth date. */
export function calcAge(isoDate: string, now: Date = new Date()): number {
  const [y, m, d] = isoDate.split('-').map(Number);
  let age = now.getFullYear() - y;
  const hadBirthday = now.getMonth() + 1 > m || (now.getMonth() + 1 === m && now.getDate() >= d);
  if (!hadBirthday) age -= 1;
  return age;
}

/** Validates three numeric text fields (day, month, year) and the 18+ rule. */
export function parseBirthDate(
  dayText: string,
  monthText: string,
  yearText: string,
  now: Date = new Date(),
): BirthDateResult {
  const day = dayText.trim();
  const month = monthText.trim();
  const year = yearText.trim();
  if (!day || !month || year.length < 4) return { ok: false, error: 'incomplete' };
  if (!/^\d{1,2}$/.test(day) || !/^\d{1,2}$/.test(month) || !/^\d{4}$/.test(year)) {
    return { ok: false, error: 'invalid' };
  }
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  const date = new Date(Date.UTC(y, m - 1, d));
  const real =
    date.getUTCFullYear() === y && date.getUTCMonth() === m - 1 && date.getUTCDate() === d;
  if (!real || y < 1900) return { ok: false, error: 'invalid' };
  const iso = `${pad(y, 4)}-${pad(m)}-${pad(d)}`;
  const age = calcAge(iso, now);
  if (age < 0 || age > 120) return { ok: false, error: 'invalid' };
  if (age < MIN_AGE) return { ok: false, error: 'underage' };
  return { ok: true, iso, age };
}

/** Splits YYYY-MM-DD into the three text fields. */
export function splitIsoDate(iso: string | null | undefined): [string, string, string] {
  const match = iso ? /^(\d{4})-(\d{2})-(\d{2})/.exec(iso) : null;
  return match ? [match[3], match[2], match[1]] : ['', '', ''];
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Keeps 18 <= min <= max <= 99 when one bound moves. */
export function adjustAgeRange(
  range: { minAge: number; maxAge: number },
  change: Partial<{ minAge: number; maxAge: number }>,
): { minAge: number; maxAge: number } {
  let minAge = clamp(change.minAge ?? range.minAge, MIN_AGE, MAX_AGE);
  let maxAge = clamp(change.maxAge ?? range.maxAge, MIN_AGE, MAX_AGE);
  if (change.minAge !== undefined && minAge > maxAge) maxAge = minAge;
  if (change.maxAge !== undefined && maxAge < minAge) minAge = maxAge;
  return { minAge, maxAge };
}

/** Distance ladder used by the stepper; `null` (last) means "anywhere". */
export const DISTANCE_STEPS: readonly (number | null)[] = [
  5,
  10,
  15,
  20,
  25,
  30,
  40,
  50,
  75,
  100,
  150,
  200,
  300,
  DISTANCE_MAX,
  null,
];

function nearestDistanceIndex(current: number | null): number {
  if (current === null) return DISTANCE_STEPS.length - 1;
  let best = 0;
  let bestDelta = Infinity;
  DISTANCE_STEPS.forEach((step, index) => {
    if (step === null) return;
    const delta = Math.abs(step - current);
    if (delta < bestDelta) {
      best = index;
      bestDelta = delta;
    }
  });
  return best;
}

export function stepDistance(current: number | null, direction: 1 | -1): number | null {
  const index = clamp(nearestDistanceIndex(current) + direction, 0, DISTANCE_STEPS.length - 1);
  return DISTANCE_STEPS[index];
}

/** Position 0..1 of the distance on the ladder, for the progress track. */
export function distanceProgress(current: number | null): number {
  return nearestDistanceIndex(current) / (DISTANCE_STEPS.length - 1);
}
