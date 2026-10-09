import {
  computeAge,
  normalizeEmail,
  parseBirthDate,
  splitIsoDate,
  validateAgeRange,
  validateBio,
  validateEmail,
  validateFirstName,
  validateMessageBody,
  validatePassword,
  validateResetCode,
} from './index';

const today = new Date(2026, 9, 9); // 9 Oct 2026

describe('parseBirthDate', () => {
  it('accepts an adult and returns ISO + age', () => {
    expect(parseBirthDate('15', '03', '1995', today)).toEqual({
      ok: true,
      iso: '1995-03-15',
      age: 31,
    });
  });
  it('pads single digits', () => {
    expect(parseBirthDate('5', '3', '1990', today)).toMatchObject({ ok: true, iso: '1990-03-05' });
  });
  it('is exactly 18 on the birthday and 17 the day before', () => {
    expect(parseBirthDate('9', '10', '2008', today)).toMatchObject({ ok: true, age: 18 });
    expect(parseBirthDate('10', '10', '2008', today)).toMatchObject({ ok: false });
  });
  it('rejects minors with the underage message', () => {
    const r = parseBirthDate('1', '1', '2015', today);
    expect(r).toEqual({ ok: false, error: expect.stringContaining('18 ans') });
  });
  it('rejects impossible dates and leap-day edge cases', () => {
    expect(parseBirthDate('31', '04', '1990', today).ok).toBe(false);
    expect(parseBirthDate('29', '02', '2001', today).ok).toBe(false);
    expect(parseBirthDate('29', '02', '2000', today).ok).toBe(true);
    expect(parseBirthDate('00', '10', '1990', today).ok).toBe(false);
    expect(parseBirthDate('10', '13', '1990', today).ok).toBe(false);
  });
  it('rejects incomplete, non numeric, future and too old values', () => {
    expect(parseBirthDate('', '10', '1990', today).ok).toBe(false);
    expect(parseBirthDate('10', '10', '90', today).ok).toBe(false);
    expect(parseBirthDate('1a', '10', '1990', today).ok).toBe(false);
    expect(parseBirthDate('10', '10', '2030', today).ok).toBe(false);
    expect(parseBirthDate('10', '10', '1900', today).ok).toBe(false);
  });
});

describe('misc validators', () => {
  it('computes age around the birthday', () => {
    expect(computeAge(2000, 10, 10, today)).toBe(25);
    expect(computeAge(2000, 10, 9, today)).toBe(26);
  });
  it('splits ISO dates', () => {
    expect(splitIsoDate('1995-03-15')).toEqual({ day: '15', month: '03', year: '1995' });
  });
  it('normalizes and validates email', () => {
    expect(normalizeEmail('  Foo@Bar.COM ')).toBe('foo@bar.com');
    expect(validateEmail(' foo@bar.com ')).toBeNull();
    expect(validateEmail('')).not.toBeNull();
    expect(validateEmail('foo@bar')).not.toBeNull();
    expect(validateEmail('foo bar@x.com')).not.toBeNull();
  });
  it('validates password length 8-128', () => {
    expect(validatePassword('1234567')).not.toBeNull();
    expect(validatePassword('12345678')).toBeNull();
    expect(validatePassword('x'.repeat(129))).not.toBeNull();
    expect(validatePassword('')).not.toBeNull();
  });
  it('validates first names (letters, accents, hyphen, apostrophe)', () => {
    expect(validateFirstName('Élodie-Anne')).toBeNull();
    expect(validateFirstName("D'Arcy")).toBeNull();
    expect(validateFirstName('')).not.toBeNull();
    expect(validateFirstName('R2D2')).not.toBeNull();
    expect(validateFirstName('a'.repeat(41))).not.toBeNull();
  });
  it('validates bio, code, message and age range', () => {
    expect(validateBio('x'.repeat(500))).toBeNull();
    expect(validateBio('x'.repeat(501))).not.toBeNull();
    expect(validateResetCode('ABCD1234')).toBeNull();
    expect(validateResetCode('ABC')).not.toBeNull();
    expect(validateMessageBody('   ')).not.toBeNull();
    expect(validateMessageBody('salut')).toBeNull();
    expect(validateMessageBody('x'.repeat(2001))).not.toBeNull();
    expect(validateAgeRange(25, 35)).toBeNull();
    expect(validateAgeRange(40, 30)).not.toBeNull();
    expect(validateAgeRange(17, 30)).not.toBeNull();
  });
});
