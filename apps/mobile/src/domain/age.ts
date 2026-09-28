export const MIN_AGE = 18;

export type BirthDateResult = { ok: true; iso: string; age: number } | { ok: false; error: string };

function pad(value: number, length: number): string {
  return String(value).padStart(length, "0");
}

export function ageFromBirthDate(iso: string, now: Date = new Date()): number {
  const [year, month, day] = iso.split("-").map(Number);
  let age = now.getFullYear() - year;
  const hadBirthday = now.getMonth() + 1 > month || (now.getMonth() + 1 === month && now.getDate() >= day);
  if (!hadBirthday) {
    age -= 1;
  }
  return age;
}

/** Validates separate day/month/year text inputs and returns an ISO date (YYYY-MM-DD). The server stays the authority. */
export function parseBirthDate(
  dayText: string,
  monthText: string,
  yearText: string,
  now: Date = new Date()
): BirthDateResult {
  const digits = /^\d+$/;
  if (!digits.test(dayText.trim()) || !digits.test(monthText.trim()) || !digits.test(yearText.trim())) {
    return { ok: false, error: "Enter your birth date using numbers only." };
  }
  const day = Number(dayText);
  const month = Number(monthText);
  const year = Number(yearText);
  if (yearText.trim().length !== 4 || year < 1900) {
    return { ok: false, error: "Enter a four digit year." };
  }
  if (month < 1 || month > 12) {
    return { ok: false, error: "Month must be between 1 and 12." };
  }
  const candidate = new Date(year, month - 1, day);
  if (candidate.getFullYear() !== year || candidate.getMonth() !== month - 1 || candidate.getDate() !== day) {
    return { ok: false, error: "That date does not exist." };
  }
  const iso = `${pad(year, 4)}-${pad(month, 2)}-${pad(day, 2)}`;
  if (candidate.getTime() > now.getTime()) {
    return { ok: false, error: "Birth date cannot be in the future." };
  }
  const age = ageFromBirthDate(iso, now);
  if (age < MIN_AGE) {
    return { ok: false, error: `You must be at least ${MIN_AGE} to use this app.` };
  }
  return { ok: true, iso, age };
}

export function splitBirthDate(iso: string): { day: string; month: string; year: string } {
  const [year = "", month = "", day = ""] = iso.split("-");
  return { day, month, year };
}
