import { useQueryClient } from "@tanstack/react-query";
import React, { useEffect, useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { updateMyLocation } from "../api/profile";
import type { Photo } from "../api/types";
import {
  buildPreferences,
  buildProfileInput,
  draftFromProfile,
  initialDraft,
  ONBOARDING_STEPS,
  validateStep,
  type Draft,
  type OnboardingStep
} from "../features/onboarding/draft";
import { InterestPicker } from "../features/profile/InterestPicker";
import { LocationField } from "../features/profile/LocationField";
import { geocodeCity } from "../features/profile/location";
import { GENDER_OPTIONS, toggleInList } from "../features/profile/options";
import { PhotoManager } from "../features/profile/PhotoManager";
import { useAuth } from "../hooks/useAuth";
import { profileKeys, useMyProfile, useSavePreferences, useSaveProfile } from "../hooks/useProfile";
import { strings } from "../i18n/strings";
import { LoadingView, reportError, showToast } from "../shared/feedback";
import { BirthDateInput, ChipGroup, ProgressBar, Stepper } from "../shared/form";
import { FormScreen } from "../shared/layout";
import { Button, Input, Text, spacing } from "../shared/ui";
import { MAX_AGE, MAX_DISTANCE_MAX_KM, MIN_AGE } from "../utils/validation";

const DISTANCE_STEP = 5;

export default function OnboardingScreen() {
  const { logout, refreshUser } = useAuth();
  const queryClient = useQueryClient();
  const profileQuery = useMyProfile();
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();

  const [step, setStep] = useState<OnboardingStep>("name");
  const [draft, setDraft] = useState<Draft>(initialDraft);
  const [showError, setShowError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [resumed, setResumed] = useState(false);

  // Resume: a profile saved in an earlier session means only photos are missing.
  useEffect(() => {
    if (resumed || profileQuery.isLoading) {
      return;
    }
    setResumed(true);
    if (profileQuery.data) {
      setDraft(draftFromProfile(profileQuery.data));
      setPhotos(profileQuery.data.photos);
      setStep("photos");
    }
  }, [profileQuery.data, profileQuery.isLoading, resumed]);

  if (!resumed) {
    return (
      <FormScreen>
        <LoadingView fullScreen />
      </FormScreen>
    );
  }

  const index = ONBOARDING_STEPS.indexOf(step);
  const error = showError ? validateStep(step, draft) : null;
  const update = (patch: Partial<Draft>) => setDraft((current) => ({ ...current, ...patch }));

  function back() {
    setShowError(false);
    setStep(ONBOARDING_STEPS[Math.max(0, index - 1)]);
  }

  async function persistProfile() {
    const input = buildProfileInput(draft);
    if (!input) {
      return false;
    }
    const saved = await saveProfile.mutateAsync(input);
    // Preferences are only seeded on first creation; later edits happen in the Profile tab.
    if (!profileQuery.data) {
      await savePreferences.mutateAsync(buildPreferences(draft));
    }
    let coords = draft.coords;
    if (!coords && draft.city.trim()) {
      coords = await geocodeCity(draft.city.trim());
    }
    if (coords) {
      try {
        await updateMyLocation(coords.latitude, coords.longitude);
      } catch (locationError) {
        reportError(locationError);
      }
    }
    setPhotos(saved.photos);
    return true;
  }

  async function next() {
    if (validateStep(step, draft)) {
      setShowError(true);
      return;
    }
    setShowError(false);
    if (step === "about") {
      setBusy(true);
      try {
        if (await persistProfile()) {
          setStep("photos");
        }
      } catch (saveError) {
        reportError(saveError);
      } finally {
        setBusy(false);
      }
      return;
    }
    setStep(ONBOARDING_STEPS[index + 1]);
  }

  async function finish() {
    if (photos.length === 0) {
      showToast(strings.onboarding.photoRequired, "error");
      return;
    }
    setBusy(true);
    try {
      const user = await refreshUser();
      if (!user.profileComplete) {
        showToast(strings.onboarding.photoRequired, "error");
      }
    } catch (finishError) {
      reportError(finishError);
    } finally {
      setBusy(false);
    }
  }

  function onPhotosChange(next: Photo[]) {
    setPhotos(next);
    queryClient.setQueryData(profileKeys.me, (current: ReturnType<typeof useMyProfile>["data"]) =>
      current ? { ...current, photos: next, complete: next.length > 0 } : current
    );
  }

  const isPhotos = step === "photos";

  return (
    <FormScreen
      testID="onboarding-screen"
      footer={
        <View style={styles.footer}>
          {isPhotos ? (
            <Button
              testID="onboarding-finish-button"
              label={strings.onboarding.finish}
              loading={busy}
              onPress={() => void finish()}
              size="lg"
            />
          ) : (
            <Button
              testID="onboarding-next-button"
              label={step === "about" ? strings.common.save : strings.common.continue}
              loading={busy}
              onPress={() => void next()}
              size="lg"
            />
          )}
        </View>
      }
    >
      <View style={styles.top}>
        {index > 0 && !(isPhotos && !profileQuery.data) ? (
          <Pressable accessibilityRole="button" hitSlop={10} onPress={back}>
            <Text tone="primary" weight="semibold">
              {strings.common.back}
            </Text>
          </Pressable>
        ) : (
          <View />
        )}
        <Pressable accessibilityRole="button" hitSlop={10} onPress={() => void logout()}>
          <Text tone="muted" weight="semibold">
            {strings.profile.logout}
          </Text>
        </Pressable>
      </View>
      <ProgressBar current={index + 1} total={ONBOARDING_STEPS.length} />

      <Text variant="title" weight="bold">
        {strings.onboarding.titles[step]}
      </Text>
      <Text tone="muted">{strings.onboarding.subtitles[step]}</Text>

      {step === "name" ? (
        <Input
          autoCapitalize="words"
          autoFocus
          error={error}
          label={strings.onboarding.firstName}
          maxLength={40}
          onChangeText={(firstName) => update({ firstName })}
          value={draft.firstName}
        />
      ) : null}

      {step === "birthDate" ? (
        <BirthDateInput error={error} onChange={(birth) => update({ birth })} value={draft.birth} />
      ) : null}

      {step === "gender" ? (
        <View style={styles.gap}>
          <ChipGroup
            onToggle={(gender) => update({ gender })}
            options={GENDER_OPTIONS}
            selected={draft.gender ? [draft.gender] : []}
          />
          {error ? (
            <Text tone="danger" variant="caption">
              {error}
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === "preferences" ? (
        <View style={styles.gapLarge}>
          <View style={styles.gap}>
            <Text tone="secondary" variant="label" weight="semibold">
              {strings.onboarding.interestedIn}
            </Text>
            <ChipGroup
              onToggle={(gender) => update({ interestedIn: toggleInList(draft.interestedIn, gender) })}
              options={GENDER_OPTIONS}
              selected={draft.interestedIn}
            />
          </View>
          <Stepper
            label={strings.onboarding.minAge}
            max={draft.maxAge}
            min={MIN_AGE}
            onChange={(minAge) => update({ minAge })}
            value={draft.minAge}
          />
          <Stepper
            label={strings.onboarding.maxAge}
            max={MAX_AGE}
            min={draft.minAge}
            onChange={(maxAge) => update({ maxAge })}
            value={draft.maxAge}
          />
          <Stepper
            label={strings.onboarding.maxDistance}
            max={MAX_DISTANCE_MAX_KM}
            min={DISTANCE_STEP}
            onChange={(maxDistanceKm) => update({ maxDistanceKm })}
            step={DISTANCE_STEP}
            unit=" km"
            value={draft.maxDistanceKm}
          />
          {error ? (
            <Text tone="danger" variant="caption">
              {error}
            </Text>
          ) : null}
        </View>
      ) : null}

      {step === "location" ? (
        <LocationField
          city={draft.city}
          cityError={error}
          coords={draft.coords}
          onCity={(city) => update({ city })}
          onCoords={(coords) => update({ coords })}
        />
      ) : null}

      {step === "about" ? (
        <View style={styles.gapLarge}>
          <Input
            error={error}
            helperText={strings.profile.bioCount(draft.bio.length, 500)}
            label={strings.profile.bio}
            maxLength={500}
            multiline
            onChangeText={(bio) => update({ bio })}
            placeholder={strings.profile.bioPlaceholder}
            value={draft.bio}
          />
          <View style={styles.gap}>
            <Text tone="secondary" variant="label" weight="semibold">
              {strings.profile.interests}
            </Text>
            <InterestPicker onChange={(interests) => update({ interests })} selected={draft.interests} />
          </View>
        </View>
      ) : null}

      {isPhotos ? <PhotoManager onChange={onPhotosChange} photos={photos} /> : null}
    </FormScreen>
  );
}

const styles = StyleSheet.create({
  top: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", minHeight: 28 },
  footer: { gap: spacing.sm },
  gap: { gap: spacing.sm },
  gapLarge: { gap: spacing.xl }
});
