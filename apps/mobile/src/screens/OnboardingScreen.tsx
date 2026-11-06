import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/errors";
import { photoApi, profileApi } from "../api/services";
import type { Gender, Photo } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { BasicsFields, basicsErrors, InterestPicker, LocationBlock, type BasicsDraft } from "../components/ProfileForm";
import { PhotoManager } from "../components/PhotoGrid";
import { Chip, Button, Input, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { locateDevice } from "../lib/device";
import { parseBirthDate } from "../lib/format";
import { spacing, useTheme } from "../theme";

const TOTAL = 5;
const GENDER_ORDER: Gender[] = ["woman", "man", "nonbinary"];

export function OnboardingScreen() {
  const { colors } = useTheme();
  const auth = useAuth();
  const queryClient = useQueryClient();
  const [step, setStep] = useState(1);
  const [showErrors, setShowErrors] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [basics, setBasics] = useState<BasicsDraft>({ firstName: "", birthDate: "", gender: null });
  const [place, setPlace] = useState<{ latitude: number; longitude: number; city: string } | null>(null);
  const [locating, setLocating] = useState(false);
  const [denied, setDenied] = useState(false);
  const [bio, setBio] = useState("");
  const [interests, setInterests] = useState<string[]>([]);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(["woman", "man", "nonbinary"]);
  const [ageRange, setAgeRange] = useState<[number, number]>([21, 45]);
  const [distance, setDistance] = useState(50);
  const [photos, setPhotos] = useState<Photo[]>([]);

  const interestsQuery = useQuery({ queryKey: ["interests"], queryFn: profileApi.interests, staleTime: Infinity });

  const saveAll = useMutation({
    mutationFn: async () => {
      const birthDate = parseBirthDate(basics.birthDate);
      if (!birthDate || !basics.gender || !place) throw new Error("incomplete");
      await profileApi.save({
        firstName: basics.firstName.trim(),
        birthDate,
        gender: basics.gender,
        bio: bio.trim(),
        city: place.city,
        latitude: place.latitude,
        longitude: place.longitude,
        interests
      });
      await profileApi.savePreferences({
        interestedIn,
        minAge: ageRange[0],
        maxAge: ageRange[1],
        maxDistanceKm: distance
      });
    }
  });

  const refreshPhotos = async () => setPhotos(await photoApi.list());

  const locate = async () => {
    setLocating(true);
    setDenied(false);
    const result = await locateDevice();
    setLocating(false);
    if (result) setPlace(result);
    else setDenied(true);
  };

  const next = async () => {
    setError(null);
    if (step === 1) {
      setShowErrors(true);
      if (Object.keys(basicsErrors(basics)).length > 0) return;
      setShowErrors(false);
    }
    if (step === 2 && !place) return setError(fr.onboarding.locationDenied);
    if (step === 4) {
      try {
        await saveAll.mutateAsync();
        await refreshPhotos();
      } catch (e) {
        return setError(errorMessage(e));
      }
    }
    setStep((s) => s + 1);
  };

  const finish = async () => {
    setError(null);
    const list = await photoApi.list().catch(() => photos);
    if (list.length === 0) return setError(fr.onboarding.needPhoto);
    await queryClient.invalidateQueries({ queryKey: ["me", "profile"] });
  };

  const toggleGender = (g: Gender) =>
    setInterestedIn((cur) => (cur.includes(g) ? (cur.length > 1 ? cur.filter((x) => x !== g) : cur) : [...cur, g]));

  return (
    <Screen>
      <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <ScrollView contentContainerStyle={{ padding: spacing.xl, gap: spacing.lg }} keyboardShouldPersistTaps="handled">
          <View style={{ maxWidth: 520, width: "100%", alignSelf: "center", gap: spacing.lg }}>
            <Text variant="caption" muted testID="onboarding-step">
              {fr.onboarding.stepOf(step, TOTAL)}
            </Text>
            <View style={{ flexDirection: "row", gap: 6 }}>
              {Array.from({ length: TOTAL }, (_, i) => (
                <View
                  key={i}
                  style={{ flex: 1, height: 4, borderRadius: 2, backgroundColor: i < step ? colors.primary : colors.border }}
                />
              ))}
            </View>

            {step === 1 ? (
              <>
                <Text variant="title">{fr.onboarding.basicsTitle}</Text>
                <BasicsFields draft={basics} onChange={setBasics} showErrors={showErrors} />
              </>
            ) : null}

            {step === 2 ? (
              <>
                <Text variant="title">{fr.onboarding.locationTitle}</Text>
                <Text muted>{fr.onboarding.locationIntro}</Text>
                <LocationBlock
                  city={place?.city ?? ""}
                  hasLocation={!!place}
                  busy={locating}
                  denied={denied}
                  onLocate={() => void locate()}
                  onPickCity={(c) => {
                    setDenied(false);
                    setPlace({ latitude: c.latitude, longitude: c.longitude, city: c.name });
                  }}
                />
              </>
            ) : null}

            {step === 3 ? (
              <>
                <Text variant="title">{fr.onboarding.aboutTitle}</Text>
                <Input
                  testID="about-bio"
                  label={fr.onboarding.bio}
                  hint={fr.onboarding.bioHint}
                  value={bio}
                  onChangeText={setBio}
                  multiline
                  maxLength={500}
                />
                <Text variant="label">{fr.onboarding.interests}</Text>
                {interestsQuery.isLoading ? (
                  <Loader />
                ) : (
                  <InterestPicker all={interestsQuery.data ?? []} selected={interests} onChange={setInterests} />
                )}
              </>
            ) : null}

            {step === 4 ? (
              <>
                <Text variant="title">{fr.onboarding.prefsTitle}</Text>
                <Text variant="label">{fr.onboarding.interestedIn}</Text>
                <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
                  {GENDER_ORDER.map((g) => (
                    <Chip
                      key={g}
                      testID={`seek-${g}`}
                      label={fr.onboarding.interestedOptions[g] ?? g}
                      selected={interestedIn.includes(g)}
                      onPress={() => toggleGender(g)}
                    />
                  ))}
                </View>
                <Stepper
                  testID="age-min"
                  label={`${fr.onboarding.ageRange} : ${ageRange[0]} – ${ageRange[1]} ans`}
                  onMinus={() => setAgeRange(([a, b]) => [Math.max(18, a - 1), b])}
                  onPlus={() => setAgeRange(([a, b]) => [Math.min(b, a + 1), b])}
                  minusLabel="Âge minimum -"
                  plusLabel="Âge minimum +"
                />
                <Stepper
                  testID="age-max"
                  label={`Âge maximum : ${ageRange[1]} ans`}
                  onMinus={() => setAgeRange(([a, b]) => [a, Math.max(a, b - 1)])}
                  onPlus={() => setAgeRange(([a, b]) => [a, Math.min(99, b + 1)])}
                  minusLabel="Âge maximum -"
                  plusLabel="Âge maximum +"
                />
                <Stepper
                  testID="distance"
                  label={`${fr.onboarding.maxDistance} : ${distance} km`}
                  onMinus={() => setDistance((d) => Math.max(1, d <= 10 ? d - 1 : d - 5))}
                  onPlus={() => setDistance((d) => Math.min(500, d < 10 ? d + 1 : d + 5))}
                  minusLabel="Distance -"
                  plusLabel="Distance +"
                />
              </>
            ) : null}

            {step === 5 ? (
              <>
                <Text variant="title">{fr.onboarding.photosTitle}</Text>
                <Text muted>{fr.onboarding.photosIntro}</Text>
                <PhotoManager photos={photos} onChanged={() => void refreshPhotos()} />
              </>
            ) : null}

            {error ? (
              <Text testID="onboarding-error" color={colors.danger} accessibilityRole="alert">
                {error}
              </Text>
            ) : null}

            {step < TOTAL ? (
              <Button testID="onboarding-next" title={fr.common.continue} loading={saveAll.isPending} onPress={() => void next()} />
            ) : (
              <Button testID="onboarding-finish" title={fr.onboarding.finish} onPress={() => void finish()} />
            )}
            {step > 1 && step < TOTAL ? (
              <Button variant="ghost" title={fr.common.back} onPress={() => setStep((s) => s - 1)} />
            ) : (
              <Button variant="ghost" title={fr.profile.logout} onPress={() => void auth.logout()} />
            )}
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </Screen>
  );
}

export function Stepper({
  label,
  onMinus,
  onPlus,
  minusLabel,
  plusLabel,
  testID
}: {
  label: string;
  onMinus: () => void;
  onPlus: () => void;
  minusLabel: string;
  plusLabel: string;
  testID?: string;
}) {
  return (
    <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md }}>
      <Text variant="label" testID={testID ? `${testID}-label` : undefined} style={{ flex: 1 }}>
        {label}
      </Text>
      <View style={{ flexDirection: "row", gap: spacing.sm }}>
        <Chip testID={testID ? `${testID}-minus` : undefined} label="−" accessibilityLabel={minusLabel} onPress={onMinus} />
        <Chip testID={testID ? `${testID}-plus` : undefined} label="+" accessibilityLabel={plusLabel} onPress={onPlus} />
      </View>
    </View>
  );
}
