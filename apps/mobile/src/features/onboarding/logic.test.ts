import type { OwnProfile } from '../../api/types';
import {
  emptyForm,
  deriveInitialStep,
  formFromProfile,
  toProfileInput,
  validateStep,
} from './logic';

const NOW = new Date(2026, 5, 15);

const own: OwnProfile = {
  userId: 'u',
  firstName: 'Camille',
  age: 31,
  gender: 'woman',
  bio: 'Salut',
  city: 'Lyon',
  interests: [{ id: 3, slug: 'x', label: 'Cinéma' }],
  photos: [],
  birthDate: '1995-03-05',
  discoverable: true,
  showDistance: true,
  hasLocation: false,
  preferences: { interestedIn: ['man'], minAge: 25, maxAge: 40, maxDistanceKm: 30 },
};

describe('resume logic', () => {
  it('starts at the beginning without a profile (GET /me/profile 404)', () => {
    expect(deriveInitialStep(null)).toBe('identity');
  });

  it('jumps to photos once a profile exists', () => {
    expect(deriveInitialStep(own)).toBe('photos');
  });

  it('hydrates the form from the saved profile', () => {
    const form = formFromProfile(own);
    expect(form).toMatchObject({
      firstName: 'Camille',
      birth: { day: '05', month: '03', year: '1995' },
      gender: 'woman',
      city: 'Lyon',
      interestIds: [3],
    });
    expect(toProfileInput(form, NOW)).toEqual({
      firstName: 'Camille',
      birthDate: '1995-03-05',
      gender: 'woman',
      bio: 'Salut',
      city: 'Lyon',
      interestIds: [3],
    });
  });
});

describe('validateStep', () => {
  it('requires name, a valid 18+ birth date and gender', () => {
    expect(Object.keys(validateStep('identity', emptyForm(), NOW)).sort()).toEqual([
      'birth',
      'firstName',
      'gender',
    ]);
    const underage = {
      ...emptyForm(),
      firstName: 'A',
      gender: 'man' as const,
      birth: { day: '16', month: '06', year: '2008' },
    };
    expect(validateStep('identity', underage, NOW)).toHaveProperty('birth');
    const ok = { ...underage, birth: { day: '15', month: '06', year: '2008' } };
    expect(validateStep('identity', ok, NOW)).toEqual({});
  });

  it('requires at least one "interested in" and a city', () => {
    expect(validateStep('preferences', emptyForm(), NOW)).toHaveProperty('interestedIn');
    expect(validateStep('location', emptyForm(), NOW)).toHaveProperty('city');
    expect(validateStep('bio', emptyForm(), NOW)).toEqual({});
    expect(validateStep('interests', emptyForm(), NOW)).toEqual({});
  });

  it('refuses to build a profile body from an invalid form', () => {
    expect(toProfileInput(emptyForm(), NOW)).toBeNull();
  });
});
