import React, { useEffect, useMemo, useState } from "react";
import { BackHandler, KeyboardAvoidingView, Platform, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import type { Gender, Me, Preferences } from "../api/types";
import { parseBirthDate, splitBirthDate } from "../domain/age";
import { GENDER_LABEL } from "../domain/labels";
import { BirthDateInput, type BirthDateParts } from "../components/BirthDateInput";
import { GenderChips } from "../components/GenderChips";
import { InterestPicker } from "../components/InterestPicker";
import { LocationCard } from "../components/LocationCard";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { useAuth, useMe } from "../hooks/useAuth";
import { useInterests, useSavePreferences, useSaveProfile } from "../hooks/useProfile";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, FormField, Input, Notice, Text } from "../shared/ui";
import { radius, spacing, useTheme } from "../theme";

const STEPS = ["name", "birth", "gender", "preferences", "about", "location", "photos"] as const;
type Step = (typeof STEPS)[number];

const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: ["man", "woman", "non_binary"],
  minAge: 18,
  maxAge: 99,
  maxDistanceKm: 50
};

function initialStep(me: Me): number {
  if (!me.profile) {
    return 0;
  }
  return STEPS.indexOf(me.profile.hasLocation ? "photos" : "location");
}

const TITLES: Record<Step, { title: string; subtitle: string }> = {
  name: { title: "What should we call you?", subtitle: "Your first name is shown on your profile." },
  birth: {
    title: "When were you born?",
    subtitle: "You must be 18 or older. We show your age, never your birth date."
  },
  gender: { title: "How do you identify?", subtitle: "Pick the option that fits you best." },
  preferences: { title: "Who would you like to meet?", subtitle: "You can change this any time." },
  about: { title: "Tell people about you", subtitle: "A few honest lines and the things you love." },
  location: { title: "Where are you?", subtitle: "We use your location to show you people nearby." },
  photos: { title: "Add your best photos", subtitle: "Clear, recent photos of you get the most conversations." }
};

function ProgressBar({ current, total }: { current: number; total: number }) {
  const theme = useTheme();
  return (
    <View
      accessibilityLabel={`Step ${current} of ${total}`}
      accessibilityRole="progressbar"
      accessibilityValue={{ min: 0, max: total, now: current }}
      style={[styles.track, { backgroundColor: theme.surfaceAlt }]}
    >
      <View style={[styles.fill, { backgroundColor: theme.primary, width: `${(current / total) * 100}%` }]} />
    </View>
  );
}

function OnboardingForm({ me }: { me: Me }) {
  const { signOut } = useAuth();
  const interestsQuery = useInterests();
  const saveProfile = useSaveProfile();
  const savePreferences = useSavePreferences();
  const profile = me.profile;
  const existingBirth = profile ? splitBirthDate(profile.birthDate) : { day: "", month: "", year: "" };

  const [step, setStep] = useState(() => initialStep(me));
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birth, setBirth] = useState<BirthDateParts>(existingBirth);
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  const [preferences, setPreferences] = useState<Preferences>(me.preferences ?? DEFAULT_PREFERENCES);
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(profile?.interests.map((interest) => interest.id) ?? []);
  const [error, setError] = useState<string | null>(null);

  const current = STEPS[step];
  const birthResult = useMemo(() => parseBirthDate(birth.day, birth.month, birth.year), [birth]);
  const saving = saveProfile.isPending || savePreferences.isPending;

  const back = () => {
    setError(null);
    setStep((value) => Math.max(0, value - 1));
  };

  useEffect(() => {
    const subscription = BackHandler.addEventListener("hardwareBackPress", () => {
      if (step > 0) {
        setError(null);
        setStep((value) => Math.max(0, value - 1));
        return true;
      }
      return false;
    });
    return () => subscription.remove();
  }, [step]);

  const next = () => setStep((value) => Math.min(STEPS.length - 1, value + 1));

  const submitAbout = async () => {
    if (!gender || !birthResult.ok) {
      setError("Please go back and complete the earlier steps.");
      return;
    }
    try {
      await savePreferences.mutateAsync(preferences);
      await saveProfile.mutateAsync({
        firstName: firstName.trim(),
        birthDate: birthResult.iso,
        gender,
        bio: bio.trim(),
        interestIds,
        showDistance: profile?.showDistance ?? true,
        isDiscoverable: profile?.isDiscoverable ?? true
      });
      setError(null);
      next();
    } catch (failure) {
      setError(errorMessage(failure, "We could not save your profile."));
    }
  };

  const onContinue = () => {
    setError(null);
    switch (current) {
      case "name":
        if (firstName.trim().length === 0 || firstName.trim().length > 50) {
          setError("Enter your first name (up to 50 characters).");
          return;
        }
        next();
        return;
      case "birth":
        if (!profile && !birthResult.ok) {
          setError(birthResult.error);
          return;
        }
        next();
        return;
      case "gender":
        if (!gender) {
          setError("Choose one option to continue.");
          return;
        }
        next();
        return;
      case "preferences":
        if (preferences.interestedIn.length === 0) {
          setError("Choose at least one option.");
          return;
        }
        next();
        return;
      case "about":
        void submitAbout();
        return;
      default:
        return;
    }
  };

  return (
    <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
      <ScreenContainer scroll>
        <View style={styles.top}>
          {step > 0 ? <Button label="Back" onPress={back} variant="ghost" /> : <View />}
          <Button label="Sign out" onPress={() => void signOut()} variant="ghost" />
        </View>
        <ProgressBar current={step + 1} total={STEPS.length} />
        <View style={styles.heading}>
          <Text variant="title">{TITLES[current].title}</Text>
          <Text tone="muted">{TITLES[current].subtitle}</Text>
        </View>
        {error ? <Notice kind="error" message={error} /> : null}

        {current === "name" ? (
          <Input
            autoCapitalize="words"
            autoComplete="given-name"
            label="First name"
            maxLength={50}
            onChangeText={setFirstName}
            value={firstName}
          />
        ) : null}
        {current === "birth" ? (
          profile ? (
            <Text>Your birth date is already set and cannot be changed.</Text>
          ) : (
            <BirthDateInput
              error={birth.day && birth.month && birth.year.length === 4 && !birthResult.ok ? birthResult.error : null}
              onChange={setBirth}
              value={birth}
            />
          )
        ) : null}
        {current === "gender" ? (
          <GenderChips labels={GENDER_LABEL} onToggle={(value) => setGender(value)} selected={gender ? [gender] : []} />
        ) : null}
        {current === "preferences" ? <PreferencesForm onChange={setPreferences} value={preferences} /> : null}
        {current === "about" ? (
          <>
            <Input
              hint={`${bio.length}/500`}
              label="About you"
              maxLength={500}
              multiline
              onChangeText={setBio}
              style={styles.bio}
              value={bio}
            />
            <FormField label="Interests">
              {interestsQuery.isPending ? (
                <LoadingView label="Loading interests" />
              ) : interestsQuery.isError ? (
                <ErrorView message={errorMessage(interestsQuery.error)} onRetry={() => void interestsQuery.refetch()} />
              ) : (
                <InterestPicker interests={interestsQuery.data} onChange={setInterestIds} selectedIds={interestIds} />
              )}
            </FormField>
          </>
        ) : null}
        {current === "location" ? (
          <LocationCard currentLabel={profile?.hasLocation ? profile.locationLabel : undefined} onSaved={next} />
        ) : null}
        {current === "photos" ? <PhotoManager photos={profile?.photos ?? []} /> : null}

        {current === "photos" ? (
          <Text center tone="muted" variant="caption">
            Your profile opens as soon as your first photo is uploaded.
          </Text>
        ) : current !== "location" ? (
          <Button
            label={current === "about" ? "Save and continue" : "Continue"}
            loading={saving}
            onPress={onContinue}
          />
        ) : null}
      </ScreenContainer>
    </KeyboardAvoidingView>
  );
}

export default function OnboardingScreen() {
  const meQuery = useMe();
  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (meQuery.isError || !meQuery.data) {
    return <ErrorView message={errorMessage(meQuery.error)} onRetry={() => void meQuery.refetch()} />;
  }
  return <OnboardingForm me={meQuery.data} />;
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  top: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  heading: { gap: spacing.sm },
  track: { height: 6, borderRadius: radius.pill, overflow: "hidden" },
  fill: { height: 6, borderRadius: radius.pill },
  bio: { minHeight: 120, textAlignVertical: "top" }
});
