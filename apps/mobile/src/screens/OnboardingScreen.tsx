import React, { useEffect, useRef, useState } from "react";
import { Animated, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getPreferences, saveProfile, savePreferences } from "../api";
import type { Gender, Preferences } from "../api/types";
import {
  BirthDateFields,
  GenderMultiPicker,
  GenderPicker,
  InterestPicker,
  LocationField
} from "../features/ProfileFields";
import { PhotoManager } from "../features/PhotoManager";
import { errorMessage } from "../i18n/errorMessage";
import { t } from "../i18n/fr";
import {
  draftFromProfile,
  draftToInput,
  emptyDraft,
  validateDraft,
  type DraftErrors,
  type ProfileDraft
} from "../lib/profileDraft";
import { validateBio, validateCity } from "../lib/validation";
import { usePhotos, useProfile } from "../queries";
import { queryKeys } from "../session/queryKeys";
import { useSession } from "../session/SessionProvider";
import { createStyles } from "../theme/ThemeProvider";
import { Button, FadeIn, LoadingState, Screen, Text, TextField, useToast } from "../ui";

const TOTAL_STEPS = 6;
const PHOTO_STEP = 5;

const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  minAge: 18,
  maxAge: 60,
  maxDistanceKm: 100,
  discoverable: true,
  showDistance: true
};

const useStyles = createStyles(({ colors, spacing, radius }) => ({
  top: { gap: spacing.sm, paddingTop: spacing.sm },
  bar: {
    height: 6,
    borderRadius: radius.pill,
    backgroundColor: colors.surfaceAlt,
    overflow: "hidden"
  },
  fill: { height: 6, borderRadius: radius.pill, backgroundColor: colors.accent },
  row: { flexDirection: "row", gap: spacing.md },
  grow: { flex: 1 },
  stepHead: { gap: spacing.xs }
}));

export default function OnboardingScreen() {
  const styles = useStyles();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { me, signOut } = useSession();
  const hasProfile = me.data?.onboarding.hasProfile ?? false;
  const profile = useProfile(hasProfile);
  const photos = usePhotos();

  const [step, setStep] = useState(0);
  const [draft, setDraft] = useState<ProfileDraft>(emptyDraft);
  const [interestedIn, setInterestedIn] = useState<Gender[]>([]);
  const [errors, setErrors] = useState<DraftErrors & { interestedIn?: string | null }>({});
  const resumed = useRef(false);
  const progress = useRef(new Animated.Value(1 / TOTAL_STEPS)).current;

  useEffect(() => {
    Animated.timing(progress, {
      toValue: (step + 1) / TOTAL_STEPS,
      duration: 260,
      useNativeDriver: false
    }).start();
  }, [step, progress]);

  // A profile already exists (app closed before adding a photo): resume at the photo step.
  useEffect(() => {
    if (resumed.current || !profile.data) return;
    resumed.current = true;
    setDraft(draftFromProfile(profile.data));
    setStep(PHOTO_STEP);
  }, [profile.data]);

  const update = (patch: Partial<ProfileDraft>) =>
    setDraft((current) => ({ ...current, ...patch }));

  const createProfile = useMutation({
    mutationFn: async () => {
      const input = draftToInput(draft);
      if (!input) throw new Error("invalid draft");
      await saveProfile(input);
      let preferences = DEFAULT_PREFERENCES;
      try {
        preferences = await getPreferences();
      } catch {
        // first time: defaults apply
      }
      await savePreferences({
        ...preferences,
        interestedIn: interestedIn.length ? interestedIn : preferences.interestedIn
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
      void queryClient.invalidateQueries({ queryKey: queryKeys.preferences });
      setStep(PHOTO_STEP);
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const finish = useMutation({
    mutationFn: () => queryClient.invalidateQueries({ queryKey: queryKeys.me }),
    onError: (error) => toast.error(errorMessage(error))
  });

  const next = () => {
    const all = validateDraft(draft);
    if (step === 0) {
      const stepErrors = { firstName: all.firstName, birth: all.birth };
      setErrors(stepErrors);
      if (stepErrors.firstName || stepErrors.birth) return;
    } else if (step === 1) {
      const stepErrors = {
        gender: all.gender,
        interestedIn: interestedIn.length ? null : t.onboarding.interestedRequired
      };
      setErrors(stepErrors);
      if (stepErrors.gender || stepErrors.interestedIn) return;
    } else if (step === 2) {
      const city = validateCity(draft.city);
      setErrors({ city });
      if (city) return;
    } else if (step === 3) {
      const bio = validateBio(draft.bio);
      setErrors({ bio });
      if (bio) return;
    } else if (step === 4) {
      createProfile.mutate();
      return;
    }
    setErrors({});
    setStep(step + 1);
  };

  const titles = [
    [t.onboarding.aboutTitle, t.onboarding.aboutText],
    [t.onboarding.genderTitle, t.onboarding.genderText],
    [t.onboarding.cityTitle, t.onboarding.cityText],
    [t.onboarding.bioTitle, t.onboarding.bioText],
    [t.onboarding.interestsTitle, t.onboarding.interestsText],
    [t.onboarding.photosTitle, t.onboarding.photosText]
  ];

  if (hasProfile && profile.isPending) return <LoadingState />;

  const photoCount = photos.data?.length ?? 0;

  return (
    <Screen edges={["top", "bottom"]} scroll testID="onboarding-screen">
      <View style={styles.top}>
        <View style={styles.row}>
          <Text tone="muted" variant="caption" style={styles.grow}>
            {t.onboarding.step(step + 1, TOTAL_STEPS)}
          </Text>
          <Button
            compact
            label={t.onboarding.logout}
            onPress={() => void signOut()}
            variant="ghost"
          />
        </View>
        <View
          accessibilityRole="progressbar"
          accessibilityValue={{ min: 1, max: TOTAL_STEPS, now: step + 1 }}
          style={styles.bar}
        >
          <Animated.View
            style={[
              styles.fill,
              { width: progress.interpolate({ inputRange: [0, 1], outputRange: ["0%", "100%"] }) }
            ]}
          />
        </View>
      </View>

      <FadeIn key={step} style={{ gap: 20 }}>
        <View style={styles.stepHead}>
          <Text variant="title">{titles[step][0]}</Text>
          <Text tone="muted">{titles[step][1]}</Text>
        </View>

        {step === 0 ? (
          <>
            <TextField
              autoCapitalize="words"
              autoComplete="given-name"
              error={errors.firstName}
              label={t.onboarding.firstName}
              maxLength={40}
              onChangeText={(firstName) => update({ firstName })}
              testID="onboarding-first-name"
              value={draft.firstName}
            />
            <BirthDateFields
              day={draft.day}
              error={errors.birth}
              month={draft.month}
              onChange={(part, value) => update({ [part]: value })}
              year={draft.year}
            />
          </>
        ) : null}

        {step === 1 ? (
          <>
            <GenderPicker
              error={errors.gender}
              label={t.onboarding.iAm}
              onChange={(gender) => update({ gender })}
              testPrefix="onboarding-gender"
              value={draft.gender}
            />
            <GenderMultiPicker
              error={errors.interestedIn}
              label={t.onboarding.interestedIn}
              onChange={setInterestedIn}
              testPrefix="onboarding-interested"
              value={interestedIn}
            />
          </>
        ) : null}

        {step === 2 ? (
          <>
            <TextField
              autoCapitalize="words"
              error={errors.city}
              label={t.onboarding.city}
              maxLength={80}
              onChangeText={(city) => update({ city })}
              testID="onboarding-city"
              value={draft.city}
            />
            <LocationField onChange={(location) => update({ location })} value={draft.location} />
          </>
        ) : null}

        {step === 3 ? (
          <TextField
            counter={`${draft.bio.length}/500`}
            error={errors.bio}
            label={t.onboarding.bio}
            multiline
            onChangeText={(bio) => update({ bio })}
            placeholder={t.onboarding.bioPlaceholder}
            testID="onboarding-bio"
            value={draft.bio}
          />
        ) : null}

        {step === 4 ? (
          <InterestPicker onChange={(interests) => update({ interests })} value={draft.interests} />
        ) : null}

        {step === PHOTO_STEP ? <PhotoManager refreshAccount={false} /> : null}
      </FadeIn>

      <View style={styles.row}>
        {step > 0 ? (
          <View style={styles.grow}>
            <Button label={t.common.back} onPress={() => setStep(step - 1)} variant="secondary" />
          </View>
        ) : null}
        <View style={styles.grow}>
          {step < PHOTO_STEP ? (
            <Button
              label={t.common.continue}
              loading={createProfile.isPending}
              onPress={next}
              testID="onboarding-next-button"
            />
          ) : (
            <Button
              disabled={photoCount < 1}
              label={t.onboarding.finish}
              loading={finish.isPending}
              onPress={() => finish.mutate()}
              testID="onboarding-finish-button"
            />
          )}
        </View>
      </View>
      {step === PHOTO_STEP && photoCount < 1 ? (
        <Text center tone="muted" variant="caption">
          {t.onboarding.needPhoto}
        </Text>
      ) : null}
    </Screen>
  );
}
