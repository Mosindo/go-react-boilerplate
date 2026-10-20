import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { saveLocation, savePreferences, saveProfile } from "../../api/profile";
import type { Gender, Preferences } from "../../api/types";
import { spacing, useTheme } from "../../theme";
import { Button, ErrorState, Input, LoadingState, Screen, Text } from "../../ui";
import { geocodeCity, detectLocation } from "../profile/location";
import { invalidateProfileData, useInterests, useMyProfile } from "../profile/hooks";
import { PhotoManager } from "../profile/PhotoManager";
import { BioField, GenderPicker, InterestPicker, InterestedInPicker, RangeFields } from "../profile/forms";
import { useAuth } from "../auth/AuthProvider";
import {
  toBirthDate,
  validateAgeRange,
  validateBio,
  validateBirthDate,
  validateFirstName
} from "../../lib/validation";

const STEPS = ["Basics", "Who you want to meet", "Where you are", "About you", "Photos"] as const;

export function OnboardingScreen() {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const { signOut, refreshUser } = useAuth();
  const profileQuery = useMyProfile();
  const interestsQuery = useInterests();

  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [firstName, setFirstName] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const [city, setCity] = useState("");
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);

  const loaded = profileQuery.data;
  useEffect(() => {
    if (!loaded) return;
    setPrefs((current) => current ?? loaded.preferences);
    const p = loaded.profile;
    if (p) {
      setFirstName((v) => v || p.firstName);
      setGender((v) => v ?? p.gender);
      setCity((v) => v || p.city);
      setBio((v) => v || p.bio);
      setInterests((v) => (v.length ? v : p.interests.map((i) => i.slug)));
      if (p.birthDate) {
        const [y, m, d] = p.birthDate.split("-");
        setYear((v) => v || y);
        setMonth((v) => v || String(Number(m)));
        setDay((v) => v || String(Number(d)));
      }
    }
  }, [loaded]);

  if (profileQuery.isLoading || !prefs) {
    return profileQuery.isError ? (
      <ErrorState error={profileQuery.error} onRetry={() => void profileQuery.refetch()} />
    ) : (
      <LoadingState />
    );
  }

  const fail = (message: string | null) => {
    setError(message);
    return message === null;
  };

  const next = async () => {
    setError(null);
    try {
      if (step === 0) {
        const birthDate = toBirthDate(year, month, day);
        const problem =
          validateFirstName(firstName) ??
          validateBirthDate(birthDate) ??
          (gender ? null : "Choose your gender.");
        if (!fail(problem)) return;
      } else if (step === 1) {
        const problem =
          prefs.interestedIn.length === 0
            ? "Choose who you would like to meet."
            : validateAgeRange(prefs.minAge, prefs.maxAge);
        if (!fail(problem)) return;
        setBusy(true);
        await savePreferences(prefs);
      } else if (step === 2) {
        if (!fail(city.trim() ? null : "Tell us your city.")) return;
        setBusy(true);
        // Only resolved here; it is sent after the profile exists (the API needs a profile first).
        setCoords(coords ?? (await geocodeCity(city.trim())));
      } else if (step === 3) {
        const birthDate = toBirthDate(year, month, day);
        if (!birthDate || !gender) return fail("Please complete your basics first.");
        if (!fail(validateBio(bio))) return;
        setBusy(true);
        await saveProfile({
          firstName: firstName.trim(),
          birthDate,
          gender,
          bio: bio.trim(),
          city: city.trim(),
          interests
        });
        const position = coords ?? (await geocodeCity(city.trim()));
        await saveLocation(position.latitude, position.longitude);
        await qc.invalidateQueries({ queryKey: ["profile", "me"] });
      } else {
        if ((profileQuery.data?.profile?.photos.length ?? 0) < 1) return fail("Add at least one photo.");
        await invalidateProfileData(qc);
        await refreshUser();
        return;
      }
      setStep((s) => s + 1);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  };

  const locateMe = async () => {
    setError(null);
    setBusy(true);
    try {
      const found = await detectLocation();
      setCoords({ latitude: found.latitude, longitude: found.longitude });
      if (found.city) setCity(found.city);
    } catch (e) {
      setError(`${e instanceof Error ? e.message : "Location unavailable."} You can type your city instead.`);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <View style={styles.progress} accessibilityLabel={`Step ${step + 1} of ${STEPS.length}`}>
        {STEPS.map((name, index) => (
          <View
            key={name}
            style={[styles.dot, { backgroundColor: index <= step ? colors.primary : colors.border }]}
          />
        ))}
      </View>
      <Text variant="title" testID="onboarding-title">
        {STEPS[step]}
      </Text>
      <View style={{ height: spacing.lg }} />

      {step === 0 ? (
        <View>
          <Input
            label="First name"
            testID="onboarding-name"
            value={firstName}
            onChangeText={setFirstName}
            autoCapitalize="words"
          />
          <Text variant="label" style={{ marginBottom: spacing.sm }}>
            Date of birth (you must be 18+)
          </Text>
          <View style={styles.dob}>
            <View style={styles.dobSmall}>
              <Input
                label="Day"
                value={day}
                onChangeText={setDay}
                keyboardType="number-pad"
                maxLength={2}
                testID="dob-day"
              />
            </View>
            <View style={styles.dobSmall}>
              <Input
                label="Month"
                value={month}
                onChangeText={setMonth}
                keyboardType="number-pad"
                maxLength={2}
                testID="dob-month"
              />
            </View>
            <View style={styles.dobYear}>
              <Input
                label="Year"
                value={year}
                onChangeText={setYear}
                keyboardType="number-pad"
                maxLength={4}
                testID="dob-year"
              />
            </View>
          </View>
          <GenderPicker label="I am" value={gender} onChange={setGender} />
        </View>
      ) : null}

      {step === 1 ? (
        <View>
          <InterestedInPicker
            value={prefs.interestedIn}
            onChange={(interestedIn) => setPrefs({ ...prefs, interestedIn })}
          />
          <RangeFields value={prefs} onChange={setPrefs} />
        </View>
      ) : null}

      {step === 2 ? (
        <View>
          <Text tone="muted" style={{ marginBottom: spacing.lg }}>
            We use your approximate location to find people nearby. Others only ever see a rough distance,
            never where you are.
          </Text>
          <Button
            label={coords ? "Location detected ✓" : "Use my location"}
            variant="secondary"
            onPress={() => void locateMe()}
            loading={busy}
            testID="use-location-button"
          />
          <View style={{ height: spacing.lg }} />
          <Input
            label="Your city"
            testID="onboarding-city"
            value={city}
            onChangeText={(v) => {
              setCity(v);
              setCoords(null);
            }}
            autoCapitalize="words"
          />
        </View>
      ) : null}

      {step === 3 ? (
        <View>
          <BioField value={bio} onChange={setBio} />
          <InterestPicker
            options={interestsQuery.data?.interests ?? []}
            selected={interests}
            onChange={setInterests}
          />
        </View>
      ) : null}

      {step === 4 ? (
        <PhotoManager
          photos={profileQuery.data?.profile?.photos ?? []}
          onChanged={() => profileQuery.refetch().then(() => undefined)}
        />
      ) : null}

      {error ? (
        <Text tone="danger" accessibilityRole="alert" style={{ marginVertical: spacing.md }}>
          {error}
        </Text>
      ) : null}

      <View style={{ height: spacing.lg }} />
      <Button
        testID="onboarding-next"
        label={step === STEPS.length - 1 ? "Start meeting people" : "Continue"}
        onPress={() => void next()}
        loading={busy && step !== 2}
      />
      {step > 0 ? (
        <Button label="Back" variant="ghost" onPress={() => setStep((s) => s - 1)} />
      ) : (
        <Button label="Sign out" variant="ghost" onPress={() => void signOut()} />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", marginBottom: spacing.xl },
  dot: { flex: 1, height: 4, borderRadius: 2, marginRight: 4 },
  dob: { flexDirection: "row" },
  dobSmall: { flex: 1, marginRight: spacing.sm },
  dobYear: { flex: 1.4 }
});
