import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../../api/keys";
import { savePreferences, saveLocation, saveProfile } from "../../api/profile";
import type { Gender, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import {
  BioField,
  Button,
  GenderPicker,
  InterestPicker,
  PhotosManager,
  PreferencesForm,
  Screen,
  Text,
  TextField,
  useToast
} from "../../components";
import { useInterests, useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import { requestDeviceLocation, type DeviceLocation } from "../../lib/location";
import { validateAgeRange, validateBio, validateBirthDate, validateFirstName } from "../../lib/validation";
import { radii, spacing, useTheme } from "../../theme";

const STEP_TITLES = ["The basics", "Tell your story", "Where are you?", "Who do you want to meet?", "Show yourself"];

export function OnboardingScreen() {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { refreshAccount, signOut } = useAuth();
  const interests = useInterests();
  const profile = useOwnProfile();

  const [step, setStep] = useState(0);
  const [firstName, setFirstName] = useState("");
  const [birthDate, setBirthDate] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [bio, setBio] = useState("");
  const [interestIds, setInterestIds] = useState<number[]>([]);
  const [city, setCity] = useState("");
  const [coords, setCoords] = useState<DeviceLocation | null>(null);
  const [prefs, setPrefs] = useState<Preferences>({ interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 });
  const [errors, setErrors] = useState<Record<string, string | null>>({});
  const [locating, setLocating] = useState(false);

  const saveStep = useMutation({
    mutationFn: async () => {
      await saveProfile({
        firstName: firstName.trim(),
        birthDate: birthDate.trim(),
        gender: gender as Gender,
        bio: bio.trim(),
        city: city.trim() || undefined,
        interestIds,
        showDistance: true,
        discoverable: true
      });
      if (coords) {
        await saveLocation(coords.latitude, coords.longitude, city.trim() || coords.city);
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      setStep(3);
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const savePrefs = useMutation({
    mutationFn: () => savePreferences(prefs),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.preferences });
      setStep(4);
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const finish = useMutation({
    mutationFn: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      await refreshAccount();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  const useMyLocation = async (): Promise<void> => {
    setLocating(true);
    try {
      const found = await requestDeviceLocation();
      setCoords(found);
      if (found.city && !city) setCity(found.city);
      toast.show("Location saved (approximate only)", "success");
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setLocating(false);
    }
  };

  const next = (): void => {
    if (step === 0) {
      const next = { firstName: validateFirstName(firstName), birthDate: validateBirthDate(birthDate), gender: gender ? null : "Choose one option." };
      setErrors(next);
      if (!next.firstName && !next.birthDate && !next.gender) setStep(1);
    } else if (step === 1) {
      const problem = validateBio(bio);
      setErrors({ bio: problem });
      if (!problem) setStep(2);
    } else if (step === 2) {
      if (!coords && !city.trim()) {
        setErrors({ city: "Share your location or type your city." });
        return;
      }
      setErrors({});
      saveStep.mutate();
    } else if (step === 3) {
      const problem = prefs.interestedIn.length === 0 ? "Pick at least one option." : validateAgeRange(prefs.minAge, prefs.maxAge);
      setErrors({ prefs: problem });
      if (!problem) savePrefs.mutate();
    } else {
      finish.mutate();
    }
  };

  const hasPhoto = (profile.data?.photos.length ?? 0) > 0;
  const loading = saveStep.isPending || savePrefs.isPending || finish.isPending;

  return (
    <Screen scroll testID="onboarding-screen">
      <View style={{ gap: spacing.xl }}>
        <View style={{ flexDirection: "row", gap: 6 }} accessibilityLabel={`Step ${step + 1} of ${STEP_TITLES.length}`}>
          {STEP_TITLES.map((_, i) => (
            <View key={i} style={{ flex: 1, height: 4, borderRadius: radii.pill, backgroundColor: i <= step ? colors.accent : colors.border }} />
          ))}
        </View>
        <Text variant="title">{STEP_TITLES[step]}</Text>

        {step === 0 ? (
          <View style={{ gap: spacing.lg }}>
            <TextField label="First name" value={firstName} onChangeText={setFirstName} error={errors.firstName} autoComplete="given-name" testID="profile-name-input" />
            <TextField
              label="Birth date"
              value={birthDate}
              onChangeText={setBirthDate}
              error={errors.birthDate}
              hint="YYYY-MM-DD · you must be 18 or older. Only your age is shown."
              keyboardType="numbers-and-punctuation"
              maxLength={10}
              testID="profile-birthdate-input"
            />
            <GenderPicker label="I am" value={gender} onChange={setGender} testIDPrefix="profile-gender" />
            {errors.gender ? <Text tone="danger">{errors.gender}</Text> : null}
          </View>
        ) : null}

        {step === 1 ? (
          <View style={{ gap: spacing.lg }}>
            <BioField value={bio} onChange={setBio} error={errors.bio} />
            <InterestPicker interests={interests.data} loading={interests.isPending} selectedIds={interestIds} onChange={setInterestIds} />
          </View>
        ) : null}

        {step === 2 ? (
          <View style={{ gap: spacing.lg }}>
            <Text tone="muted">
              We use your location only to find people nearby. Others see an approximate distance, never your position.
            </Text>
            <Button title={coords ? "Location saved ✓" : "Use my location"} variant="secondary" onPress={useMyLocation} loading={locating} testID="onboarding-locate" />
            <TextField label="City" value={city} onChangeText={setCity} error={errors.city} hint="Shown on your profile." testID="profile-city-input" />
          </View>
        ) : null}

        {step === 3 ? (
          <View style={{ gap: spacing.lg }}>
            <PreferencesForm value={prefs} onChange={setPrefs} />
            {errors.prefs ? <Text tone="danger">{errors.prefs}</Text> : null}
          </View>
        ) : null}

        {step === 4 ? (
          <View style={{ gap: spacing.lg }}>
            <PhotosManager photos={profile.data?.photos ?? []} />
          </View>
        ) : null}

        <View style={{ gap: spacing.sm }}>
          <Button
            title={step === 4 ? "Start meeting people" : "Continue"}
            onPress={next}
            loading={loading}
            disabled={step === 4 && !hasPhoto}
            testID="onboarding-next"
          />
          {step > 0 ? (
            <Button title="Back" variant="ghost" onPress={() => setStep(step - 1)} />
          ) : null}
          <Button title="Sign out" variant="ghost" onPress={() => void signOut()} />
        </View>
      </View>
    </Screen>
  );
}
