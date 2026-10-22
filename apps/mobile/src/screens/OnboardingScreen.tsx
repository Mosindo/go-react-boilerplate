import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { saveProfile, savePreferences } from "../api/profile";
import type { Gender, Preferences } from "../api/types";
import { DEFAULT_PREFERENCES, PreferencesForm } from "../components/PreferencesForm";
import { InterestPicker } from "../components/InterestPicker";
import { PhotoManager } from "../components/PhotoManager";
import { useAuth } from "../hooks/useAuth";
import { usePhotosQuery } from "../hooks/useProfileData";
import { CITIES } from "../lib/cities";
import { LocationDeniedError, readDeviceLocation } from "../lib/location";
import { MAX_BIO, parseBirthdate, roundCoordinate } from "../lib/validation";
import { ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text, colors, spacing } from "../shared/ui";

const STEPS = ["Qui êtes-vous ?", "Qui souhaitez-vous rencontrer ?", "Où habitez-vous ?", "Vos photos", "À propos de vous"] as const;

const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "non_binary", label: "Non-binaire" }
];

type Place = { city: string; latitude: number; longitude: number };

export default function OnboardingScreen() {
  const { refreshUser, signOut } = useAuth();
  const queryClient = useQueryClient();
  const photos = usePhotosQuery();

  const [step, setStep] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [locating, setLocating] = useState(false);

  const [firstName, setFirstName] = useState("");
  const [day, setDay] = useState("");
  const [month, setMonth] = useState("");
  const [year, setYear] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [preferences, setPreferences] = useState<Preferences>(DEFAULT_PREFERENCES);
  const [place, setPlace] = useState<Place | null>(null);
  const [bio, setBio] = useState("");
  const [interestIds, setInterestIds] = useState<number[]>([]);

  const validateStep = (): string | null => {
    switch (step) {
      case 0: {
        if (!firstName.trim()) {
          return "Indiquez votre prénom.";
        }
        const birth = parseBirthdate(day, month, year);
        if (!birth.ok) {
          return birth.error;
        }
        return gender ? null : "Choisissez votre genre.";
      }
      case 1:
        return preferences.interestedIn.length === 0 ? "Choisissez au moins une option." : null;
      case 2:
        return place && place.city.trim() ? null : "Indiquez où vous habitez.";
      case 3:
        return (photos.data?.length ?? 0) === 0 ? "Ajoutez au moins une photo." : null;
      default:
        return null;
    }
  };

  const next = () => {
    const issue = validateStep();
    setError(issue);
    if (!issue) {
      setStep((value) => value + 1);
    }
  };

  const locate = async () => {
    setLocating(true);
    setError(null);
    try {
      const found = await readDeviceLocation();
      setPlace({ city: found.city, latitude: found.latitude, longitude: found.longitude });
    } catch (err) {
      setError(
        err instanceof LocationDeniedError
          ? "Position refusée : choisissez plutôt votre ville ci-dessous."
          : "Position indisponible : choisissez votre ville ci-dessous."
      );
    } finally {
      setLocating(false);
    }
  };

  const finish = async () => {
    const birth = parseBirthdate(day, month, year);
    if (!birth.ok || !gender || !place) {
      setError("Certaines informations sont manquantes.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await saveProfile({
        firstName: firstName.trim(),
        birthdate: birth.iso,
        gender,
        bio: bio.trim(),
        city: place.city.trim(),
        latitude: roundCoordinate(place.latitude),
        longitude: roundCoordinate(place.longitude),
        interestIds
      });
      await savePreferences(preferences);
      await queryClient.invalidateQueries();
      await refreshUser();
    } catch (err) {
      setError(errorMessage(err, "Enregistrement impossible."));
    } finally {
      setSaving(false);
    }
  };

  const isLast = step === STEPS.length - 1;

  return (
    <ScreenContainer contentMaxWidth={520} edges={["top", "left", "right"]} testID="onboarding-screen">
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <View style={styles.progress}>
          {STEPS.map((title, index) => (
            <View key={title} style={[styles.bar, index <= step ? styles.barActive : null]} />
          ))}
        </View>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled" showsVerticalScrollIndicator={false}>
          <Text tone="muted" variant="eyebrow" weight="bold">
            Étape {step + 1} sur {STEPS.length}
          </Text>
          <Text accessibilityRole="header" variant="title" weight="bold">
            {STEPS[step]}
          </Text>

          {step === 0 ? (
            <View style={styles.form}>
              <Input
                autoCapitalize="words"
                label="Prénom"
                maxLength={40}
                onChangeText={setFirstName}
                testID="onboarding-name"
                value={firstName}
              />
              <FormField helperText="Vous devez avoir 18 ans ou plus. Seul votre âge sera affiché." label="Date de naissance">
                <View style={styles.dateRow}>
                  <Input
                    accessibilityLabel="Jour"
                    containerStyle={styles.dateSmall}
                    keyboardType="number-pad"
                    maxLength={2}
                    onChangeText={setDay}
                    placeholder="JJ"
                    testID="onboarding-day"
                    value={day}
                  />
                  <Input
                    accessibilityLabel="Mois"
                    containerStyle={styles.dateSmall}
                    keyboardType="number-pad"
                    maxLength={2}
                    onChangeText={setMonth}
                    placeholder="MM"
                    testID="onboarding-month"
                    value={month}
                  />
                  <Input
                    accessibilityLabel="Année"
                    containerStyle={styles.dateYear}
                    keyboardType="number-pad"
                    maxLength={4}
                    onChangeText={setYear}
                    placeholder="AAAA"
                    testID="onboarding-year"
                    value={year}
                  />
                </View>
              </FormField>
              <FormField label="Genre">
                <View style={styles.chips}>
                  {GENDERS.map((option) => (
                    <Chip
                      key={option.value}
                      label={option.label}
                      onPress={() => setGender(option.value)}
                      selected={gender === option.value}
                      testID={`gender-${option.value}`}
                    />
                  ))}
                </View>
              </FormField>
            </View>
          ) : null}

          {step === 1 ? <PreferencesForm onChange={setPreferences} value={preferences} /> : null}

          {step === 2 ? (
            <View style={styles.form}>
              <Text tone="muted">
                Votre position précise n&apos;est jamais partagée : les autres voient seulement une distance approximative, et nous la
                gardons à 1 km près.
              </Text>
              <Button label="Utiliser ma position" loading={locating} onPress={locate} testID="use-location" variant="secondary" />
              <FormField label="Ou choisissez une ville">
                <View style={styles.chips}>
                  {CITIES.map((city) => (
                    <Chip
                      key={city.name}
                      label={city.name}
                      onPress={() => setPlace({ city: city.name, latitude: city.latitude, longitude: city.longitude })}
                      selected={place?.city === city.name}
                      testID={`city-${city.name}`}
                    />
                  ))}
                </View>
              </FormField>
              {place ? (
                <Input
                  label="Ville affichée"
                  maxLength={80}
                  onChangeText={(city) => setPlace({ ...place, city })}
                  testID="onboarding-city"
                  value={place.city}
                />
              ) : null}
            </View>
          ) : null}

          {step === 3 ? (
            <View style={styles.form}>
              <Text tone="muted">Ajoutez jusqu&apos;à 6 photos. La première sera votre photo principale.</Text>
              <PhotoManager />
            </View>
          ) : null}

          {step === 4 ? (
            <View style={styles.form}>
              <Input
                helperText={`${bio.length} / ${MAX_BIO}`}
                label="Bio"
                maxLength={MAX_BIO}
                multiline
                onChangeText={setBio}
                placeholder="Quelques mots sur vous…"
                testID="onboarding-bio"
                value={bio}
              />
              <FormField label="Centres d'intérêt">
                <InterestPicker onChange={setInterestIds} selected={interestIds} />
              </FormField>
            </View>
          ) : null}

          {error ? <Notice description={error} title="Un instant" tone="danger" /> : null}
        </ScrollView>

        <View style={styles.footer}>
          {step > 0 ? (
            <Button
              label="Retour"
              onPress={() => {
                setError(null);
                setStep(step - 1);
              }}
              variant="ghost"
            />
          ) : (
            <Button label="Se déconnecter" onPress={signOut} variant="ghost" />
          )}
          {isLast ? (
            <Button label="Terminer" loading={saving} onPress={finish} size="lg" testID="onboarding-finish" />
          ) : (
            <Button label="Continuer" onPress={next} size="lg" testID="onboarding-next" />
          )}
        </View>
      </KeyboardAvoidingView>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  progress: { flexDirection: "row", gap: 6, marginBottom: spacing.lg },
  bar: { flex: 1, height: 4, borderRadius: 2, backgroundColor: colors.border },
  barActive: { backgroundColor: colors.primary },
  content: { gap: spacing.lg, paddingBottom: spacing.xl },
  form: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  dateRow: { flexDirection: "row", gap: spacing.sm },
  dateSmall: { width: 78 },
  dateYear: { width: 110 },
  footer: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", paddingTop: spacing.md }
});
