import {
  adjustAgeRange,
  calcAge,
  distanceProgress,
  isValidEmail,
  isValidPassword,
  normalizeEmail,
  parseBirthDate,
  splitIsoDate,
  stepDistance,
} from './validation';

const NOW = new Date(2026, 5, 15); // 15 June 2026

describe('calcAge', () => {
  it('counts whole years and respects the birthday', () => {
    expect(calcAge('2000-06-15', NOW)).toBe(26);
    expect(calcAge('2000-06-16', NOW)).toBe(25);
    expect(calcAge('2000-01-01', NOW)).toBe(26);
    expect(calcAge('2008-06-15', NOW)).toBe(18);
    expect(calcAge('2008-06-16', NOW)).toBe(17);
  });
});

describe('parseBirthDate', () => {
  it('accepts an adult birth date and returns ISO + age', () => {
    expect(parseBirthDate('5', '3', '1995', NOW)).toEqual({ ok: true, iso: '1995-03-05', age: 31 });
  });

  it('accepts exactly 18 today and rejects 17 years 364 days', () => {
    expect(parseBirthDate('15', '06', '2008', NOW)).toMatchObject({ ok: true, age: 18 });
    expect(parseBirthDate('16', '06', '2008', NOW)).toEqual({ ok: false, error: 'underage' });
  });

  it('flags incomplete input', () => {
    expect(parseBirthDate('', '3', '1995', NOW)).toEqual({ ok: false, error: 'incomplete' });
    expect(parseBirthDate('5', '3', '19', NOW)).toEqual({ ok: false, error: 'incomplete' });
  });

  it('rejects impossible and out-of-range dates', () => {
    expect(parseBirthDate('31', '04', '1990', NOW)).toEqual({ ok: false, error: 'invalid' });
    expect(parseBirthDate('29', '02', '1999', NOW)).toEqual({ ok: false, error: 'invalid' });
    expect(parseBirthDate('29', '02', '2000', NOW)).toMatchObject({ ok: true });
    expect(parseBirthDate('01', '01', '1850', NOW)).toEqual({ ok: false, error: 'invalid' });
    expect(parseBirthDate('01', '01', '2030', NOW)).toEqual({ ok: false, error: 'invalid' });
    expect(parseBirthDate('ab', '01', '1990', NOW)).toEqual({ ok: false, error: 'invalid' });
  });
});

describe('splitIsoDate', () => {
  it('splits into day, month, year', () => {
    expect(splitIsoDate('1995-03-05')).toEqual(['05', '03', '1995']);
    expect(splitIsoDate(undefined)).toEqual(['', '', '']);
  });
});

describe('credentials', () => {
  it('normalizes and validates emails', () => {
    expect(normalizeEmail('  Foo@Bar.COM ')).toBe('foo@bar.com');
    expect(isValidEmail(' a@b.fr ')).toBe(true);
    expect(isValidEmail('a@b')).toBe(false);
    expect(isValidEmail('a b@c.fr')).toBe(false);
  });

  it('enforces the 8-72 password window', () => {
    expect(isValidPassword('1234567')).toBe(false);
    expect(isValidPassword('12345678')).toBe(true);
    expect(isValidPassword('x'.repeat(72))).toBe(true);
    expect(isValidPassword('x'.repeat(73))).toBe(false);
  });
});

describe('adjustAgeRange', () => {
  it('keeps min <= max within 18..99', () => {
    expect(adjustAgeRange({ minAge: 25, maxAge: 30 }, { minAge: 35 })).toEqual({
      minAge: 35,
      maxAge: 35,
    });
    expect(adjustAgeRange({ minAge: 25, maxAge: 30 }, { maxAge: 20 })).toEqual({
      minAge: 20,
      maxAge: 20,
    });
    expect(adjustAgeRange({ minAge: 25, maxAge: 30 }, { minAge: 10 })).toEqual({
      minAge: 18,
      maxAge: 30,
    });
    expect(adjustAgeRange({ minAge: 25, maxAge: 30 }, { maxAge: 120 })).toEqual({
      minAge: 25,
      maxAge: 99,
    });
  });
});

describe('distance stepper', () => {
  it('walks the ladder up to "anywhere" (null) and back', () => {
    expect(stepDistance(5, -1)).toBe(5);
    expect(stepDistance(25, 1)).toBe(30);
    expect(stepDistance(500, 1)).toBeNull();
    expect(stepDistance(null, 1)).toBeNull();
    expect(stepDistance(null, -1)).toBe(500);
  });

  it('snaps off-ladder server values to the nearest step', () => {
    expect(stepDistance(12, 1)).toBe(15);
    expect(distanceProgress(5)).toBe(0);
    expect(distanceProgress(null)).toBe(1);
  });
});
