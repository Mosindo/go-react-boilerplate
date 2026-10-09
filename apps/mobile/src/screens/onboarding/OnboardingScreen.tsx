import React, { useMemo, useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { profileApi } from '../../api/profile';
import {
  GENDERS,
  type Gender,
  type Me,
  type MyProfile,
  type Preferences,
  type ProfileInput,
} from '../../api/types';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import { qk } from '../../query/keys';
import { usePreferences } from '../../query/hooks';
import { BirthDateInput, type BirthDateParts } from '../../shared/forms/BirthDateInput';
import { InterestPicker } from '../../shared/forms/InterestPicker';
import { LocationPicker, type LocationValue } from '../../shared/forms/LocationPicker';
import { PhotoManager } from '../../shared/forms/PhotoManager';
import { PreferencesForm } from '../../shared/forms/PreferencesForm';
import {
  Button,
  ChoiceGroup,
  ErrorState,
  LoadingView,
  Screen,
  Text,
  TextField,
} from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { radius, spacing, useTheme } from '../../theme';
import { splitIsoDate, parseBirthDate, validateBio, validateFirstName } from '../../validators';

type Step = 'basics' | 'location' | 'photos' | 'bio' | 'preferences';
const STEPS: readonly Step[] = ['basics', 'location', 'photos', 'bio', 'preferences'];

type FormState = {
  firstName: string;
  birth: BirthDateParts;
  gender: Gender | null;
  location: LocationValue;
  bio: string;
  interests: string[];
};

export function initialStep(onboarding: Me['onboarding'], profile: MyProfile | undefined): Step {
  if (!onboarding.profile || !profile) return 'basics';
  if (!onboarding.photos) return 'photos';
  if (!profile.bio && profile.interests.length === 0) return 'bio';
  return 'preferences';
}

function formFromProfile(p: MyProfile | undefined): FormState {
  return {
    firstName: p?.firstName ?? '',
    birth: p ? splitIsoDate(p.birthDate) : { day: '', month: '', year: '' },
    gender: p?.gender ?? null,
    location: {
      city: p?.city ?? '',
      latitude: p ? p.latitude : null,
      longitude: p ? p.longitude : null,
    },
    bio: p?.bio ?? '',
    interests: p?.interests.map((i) => i.slug) ?? [],
  };
}

const genderChoices = GENDERS.map((g) => ({ value: g, label: fr.gender[g] }));

/** Resumable onboarding wizard. Shown while GET /me onboarding.complete is false. */
export function OnboardingScreen({ me }: { me: Me }) {
  const profileQuery = useQuery({
    queryKey: qk.myProfile,
    queryFn: ({ signal }) => profileApi.getProfile(signal),
    enabled: me.onboarding.profile,
    retry: false,
  });
  if (me.onboarding.profile && profileQuery.isPending) return <LoadingView />;
  if (me.onboarding.profile && profileQuery.isError) {
    return (
      <ErrorState message={fr.profile.loadFailed} onRetry={() => void profileQuery.refetch()} />
    );
  }
  return <Wizard me={me} profile={profileQuery.data} />;
}

function Wizard({ me, profile }: { me: Me; profile: MyProfile | undefined }) {
  const qc = useQueryClient();
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const [step, setStep] = useState<Step>(() => initialStep(me.onboarding, profile));
  const [form, setForm] = useState<FormState>(() => formFromProfile(profile));
  const [errors, setErrors] = useState<Record<string, string | undefined>>({});
  const prefsQuery = usePreferences(true);
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const effectivePrefs = prefs ?? prefsQuery.data ?? null;
  const index = STEPS.indexOf(step);

  const patch = (p: Partial<FormState>) => setForm((f) => ({ ...f, ...p }));
  const go = (s: Step) => {
    setErrors({});
    setStep(s);
  };

  const birth = useMemo(
    () => parseBirthDate(form.birth.day, form.birth.month, form.birth.year),
    [form.birth.day, form.birth.month, form.birth.year],
  );

  const buildInput = (): ProfileInput | null => {
    if (
      !birth.ok ||
      !form.gender ||
      form.location.latitude === null ||
      form.location.longitude === null
    ) {
      return null;
    }
    return {
      firstName: form.firstName.trim(),
      birthDate: birth.iso,
      gender: form.gender,
      bio: form.bio.trim(),
      city: form.location.city.trim(),
      latitude: form.location.latitude,
      longitude: form.location.longitude,
      interests: form.interests,
      discoverable: profile?.discoverable ?? true,
      showDistance: profile?.showDistance ?? true,
    };
  };

  const saveProfile = useMutation({
    mutationFn: async (): Promise<MyProfile> => {
      const input = buildInput();
      if (!input) throw new Error('incomplete form');
      return profileApi.saveProfile(input);
    },
    onSuccess: (saved) => {
      qc.setQueryData(qk.myProfile, saved);
      void qc.invalidateQueries({ queryKey: qk.me });
    },
  });

  const savePrefs = useMutation({
    mutationFn: (p: Preferences) => profileApi.savePreferences(p),
    onSuccess: async (saved) => {
      qc.setQueryData(qk.preferences, saved);
      await qc.invalidateQueries({ queryKey: qk.me });
    },
  });

  const nextFromBasics = () => {
    const next = {
      firstName: validateFirstName(form.firstName) ?? undefined,
      birth: birth.ok ? undefined : birth.error,
      gender: form.gender ? undefined : fr.validation.genderRequired,
    };
    setErrors(next);
    if (!next.firstName && !next.birth && !next.gender) go('location');
  };

  const nextFromLocation = () => {
    const next = {
      city: form.location.city.trim() ? undefined : fr.validation.cityRequired,
      coords: form.location.latitude === null ? fr.onboarding.cityNeedsCoords : undefined,
    };
    setErrors(next);
    if (next.city || next.coords) return;
    saveProfile.mutate(undefined, { onSuccess: () => go('photos') });
  };

  const nextFromPhotos = async () => {
    const photos = qc.getQueryData<unknown[]>(qk.myPhotos) ?? (await profileApi.listPhotos());
    if (photos.length === 0) {
      setErrors({ photos: fr.photos.minOne });
      return;
    }
    go('bio');
  };

  const nextFromBio = () => {
    const bioError = validateBio(form.bio) ?? undefined;
    setErrors({ bio: bioError });
    if (bioError) return;
    saveProfile.mutate(undefined, { onSuccess: () => go('preferences') });
  };

  const finish = () => {
    if (!effectivePrefs) return;
    if (effectivePrefs.interestedIn.length === 0) {
      setErrors({ prefs: fr.validation.interestedInRequired });
      return;
    }
    savePrefs.mutate(effectivePrefs, { onSuccess: () => toast.success(fr.app.tagline) });
  };

  const titles: Record<Step, [string, string]> = {
    basics: [fr.onboarding.basicsTitle, fr.onboarding.basicsSubtitle],
    location: [fr.onboarding.locationTitle, fr.onboarding.locationSubtitle],
    photos: [fr.onboarding.photosTitle, fr.onboarding.photosSubtitle],
    bio: [fr.onboarding.bioTitle, fr.onboarding.bioSubtitle],
    preferences: [fr.onboarding.preferencesTitle, fr.onboarding.preferencesSubtitle],
  };

  const backStep = index > 0 ? STEPS[index - 1] : null;
  const busy = saveProfile.isPending || savePrefs.isPending;

  const footer = (
    <View style={styles.footer}>
      {backStep ? (
        <Button
          label={fr.common.back}
          variant="secondary"
          onPress={() => go(backStep)}
          style={styles.back}
          testID="onboarding-back"
        />
      ) : null}
      {step === 'basics' ? (
        <Button
          label={fr.common.continue}
          onPress={nextFromBasics}
          style={styles.next}
          testID="onboarding-next"
        />
      ) : null}
      {step === 'location' ? (
        <Button
          label={fr.common.continue}
          onPress={nextFromLocation}
          loading={busy}
          style={styles.next}
          testID="onboarding-next"
        />
      ) : null}
      {step === 'photos' ? (
        <Button
          label={fr.common.continue}
          onPress={() => void nextFromPhotos()}
          style={styles.next}
          testID="onboarding-next"
        />
      ) : null}
      {step === 'bio' ? (
        <Button
          label={fr.common.continue}
          onPress={nextFromBio}
          loading={busy}
          style={styles.next}
          testID="onboarding-next"
        />
      ) : null}
      {step === 'preferences' ? (
        <Button
          label={fr.onboarding.finish}
          onPress={finish}
          loading={busy}
          disabled={!effectivePrefs}
          style={styles.next}
          testID="onboarding-finish"
        />
      ) : null}
    </View>
  );

  return (
    <Screen scroll edges={['top', 'bottom']} footer={footer} testID="onboarding-screen">
      <View style={styles.topRow}>
        <Text
          variant="label"
          tone="muted"
          accessibilityLiveRegion="polite"
          testID="onboarding-step"
        >
          {fr.onboarding.step(index + 1, STEPS.length)}
        </Text>
        <Button
          label={fr.auth.logout}
          variant="ghost"
          compact
          onPress={() => void signOut()}
          testID="onboarding-logout"
        />
      </View>
      <View
        style={styles.progress}
        accessibilityRole="progressbar"
        accessibilityValue={{ min: 1, max: STEPS.length, now: index + 1 }}
      >
        {STEPS.map((s, i) => (
          <View
            key={s}
            style={[
              styles.segment,
              { backgroundColor: i <= index ? colors.primary : colors.border },
            ]}
          />
        ))}
      </View>

      <Text variant="title" accessibilityRole="header" style={styles.title}>
        {titles[step][0]}
      </Text>
      <Text tone="muted" style={styles.subtitle}>
        {titles[step][1]}
      </Text>

      {step === 'basics' ? (
        <View>
          <TextField
            testID="onboarding-firstName"
            label={fr.onboarding.firstName}
            value={form.firstName}
            onChangeText={(firstName) => patch({ firstName })}
            error={errors.firstName}
            autoCapitalize="words"
            autoComplete="given-name"
            maxLength={40}
          />
          <Text variant="label" tone="muted" style={styles.fieldLabel}>
            {fr.onboarding.birthDate}
          </Text>
          <BirthDateInput
            value={form.birth}
            onChange={(b) => patch({ birth: b })}
            error={errors.birth}
          />
          <Text variant="label" tone="muted" style={[styles.fieldLabel, styles.genderGap]}>
            {fr.onboarding.gender}
          </Text>
          <ChoiceGroup
            mode="single"
            choices={genderChoices}
            selected={form.gender ? [form.gender] : []}
            onToggle={(gender) => patch({ gender })}
            testIDPrefix="onboarding-gender"
            label={fr.onboarding.gender}
          />
          {errors.gender ? (
            <Text variant="caption" tone="danger" accessibilityRole="alert">
              {errors.gender}
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === 'location' ? (
        <LocationPicker
          value={form.location}
          onChange={(location) => patch({ location })}
          cityError={errors.city}
          coordsError={errors.coords}
        />
      ) : null}

      {step === 'photos' ? (
        <View>
          <PhotoManager />
          {errors.photos ? (
            <Text
              tone="danger"
              accessibilityRole="alert"
              testID="photos-error"
              style={styles.fieldLabel}
            >
              {errors.photos}
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === 'bio' ? (
        <View>
          <TextField
            testID="onboarding-bio"
            label={fr.interests.bio}
            placeholder={fr.interests.bioPlaceholder}
            value={form.bio}
            onChangeText={(bio) => patch({ bio })}
            error={errors.bio}
            multiline
            counter={500}
          />
          <Text variant="heading" style={styles.fieldLabel}>
            {fr.interests.title}
          </Text>
          <InterestPicker
            selected={form.interests}
            onChange={(interests) => patch({ interests })}
          />
        </View>
      ) : null}

      {step === 'preferences' ? (
        effectivePrefs ? (
          <PreferencesForm value={effectivePrefs} onChange={setPrefs} error={errors.prefs} />
        ) : prefsQuery.isError ? (
          <ErrorState onRetry={() => void prefsQuery.refetch()} />
        ) : (
          <LoadingView />
        )
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  topRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  progress: { flexDirection: 'row', gap: spacing.xs, marginVertical: spacing.md },
  segment: { flex: 1, height: 4, borderRadius: radius.pill },
  title: { marginTop: spacing.md },
  subtitle: { marginTop: spacing.xs, marginBottom: spacing.xl },
  fieldLabel: { marginTop: spacing.sm, marginBottom: spacing.sm },
  genderGap: { marginTop: spacing.xl },
  footer: { flexDirection: 'row' },
  back: { marginRight: spacing.md },
  next: { flex: 1 },
});
