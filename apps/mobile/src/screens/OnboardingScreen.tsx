import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { Gender, OwnProfile, Preferences } from "../api/types";
import { InterestPicker } from "../components/InterestPicker";
import { PhotoGrid } from "../components/PhotoGrid";
import { DEFAULT_PREFERENCES, PreferencesForm, validatePreferences } from "../components/PreferencesForm";
import { errorMessage, keys } from "../hooks/queries";
import { readCoarsePosition, type CoarsePosition } from "../hooks/useLocation";
import { formatBirthDateInput } from "../lib/dates";
import { validateBirthDate, validateDisplayName } from "../lib/validation";
import { useSession } from "../state/session";
import { Button, Chip, Input, Screen, Text, radii, spacing, useTheme } from "../ui";

type Step = "identity" | "location" | "preferences" | "photos" | "about";
const ORDER: Step[] = ["identity", "location", "preferences", "photos", "about"];

const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Une femme" },
  { value: "man", label: "Un homme" },
  { value: "non_binary", label: "Non-binaire" }
];

function firstStep(profile: OwnProfile | null): Step {
  if (!profile) return "identity";
  if (profile.missing.includes("location")) return "location";
  if (profile.missing.includes("preferences")) return "preferences";
  return "photos";
}

type Props = { profile: OwnProfile | null; onFinish: () => void };

export default function OnboardingScreen({ profile, onFinish }: Props) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const { logout } = useSession();
  const [step, setStep] = useState<Step>(() => firstStep(profile));
  const [name, setName] = useState(profile?.displayName ?? "");
  const [birthText, setBirthText] = useState("");
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  const [position, setPosition] = useState<CoarsePosition | null>(null);
  const [city, setCity] = useState(profile?.city ?? "");
  const [prefs, setPrefs] = useState<Preferences>(
    profile?.preferences.interestedIn.length ? profile.preferences : DEFAULT_PREFERENCES
  );
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [interests, setInterests] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [birthIso, setBirthIso] = useState<string | null>(profile?.birthDate ?? null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => setError(null), [step]);

  const saveProfile = async (extra: { latitude?: number; longitude?: number }) => {
    if (!birthIso || !gender) return;
    await profileApi.save({
      displayName: name.trim(),
      birthDate: birthIso,
      gender,
      bio: bio.trim(),
      city: city.trim(),
      interests,
      ...extra
    });
  };

  const run = async (action: () => Promise<void>, next?: Step) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      if (next) setStep(next);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const finish = async () => {
    await queryClient.invalidateQueries({ queryKey: keys.profile });
    onFinish();
  };

  const onIdentity = () => {
    const nameError = validateDisplayName(name);
    const birth = validateBirthDate(birthText);
    if (nameError || birth.error) return setError(nameError ?? birth.error);
    if (!gender) return setError("Précisez votre genre.");
    setBirthIso(birth.iso);
    setStep("location");
  };

  const locate = () =>
    run(async () => {
      const found = await readCoarsePosition();
      setPosition(found);
      if (found.city) setCity(found.city);
    });

  const onLocation = () => {
    if (!position && !profile?.hasLocation) return setError("Activez la localisation pour continuer.");
    void run(async () => {
      await saveProfile(position ? { latitude: position.latitude, longitude: position.longitude } : {});
    }, "preferences");
  };

  const onPreferences = () => {
    const invalid = validatePreferences(prefs);
    if (invalid) return setError(invalid);
    void run(async () => {
      await profileApi.savePreferences(prefs);
    }, "photos");
  };

  const onPhotos = async () => {
    setBusy(true);
    try {
      const fresh = await profileApi.get();
      if (fresh.photos.length === 0) return setError("Ajoutez au moins une photo.");
      setStep("about");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const onAbout = () =>
    void run(async () => {
      await saveProfile({});
      await finish();
    });

  const index = ORDER.indexOf(step);

  return (
    <Screen scroll testID="onboarding-screen" topInset>
      <View style={styles.progress} accessibilityLabel={`Étape ${index + 1} sur ${ORDER.length}`}>
        {ORDER.map((s, i) => (
          <View key={s} style={[styles.dot, { backgroundColor: i <= index ? colors.primary : colors.border }]} />
        ))}
      </View>

      {step === "identity" ? (
        <View style={styles.step}>
          <Text variant="title">Faisons connaissance</Text>
          <Input
            autoCapitalize="words"
            label="Prénom"
            maxLength={50}
            onChangeText={setName}
            testID="ob-name"
            value={name}
          />
          <Input
            hint="Seul votre âge sera visible."
            keyboardType="number-pad"
            label="Date de naissance"
            maxLength={10}
            onChangeText={(v) => setBirthText(formatBirthDateInput(v))}
            placeholder="JJ/MM/AAAA"
            testID="ob-birth"
            value={birthText}
          />
          <Text variant="label" tone="muted">
            Je suis
          </Text>
          <View style={styles.chips}>
            {GENDERS.map((g) => (
              <Chip
                key={g.value}
                label={g.label}
                onPress={() => setGender(g.value)}
                selected={gender === g.value}
                testID={`ob-gender-${g.value}`}
              />
            ))}
          </View>
          <Button label="Continuer" onPress={onIdentity} testID="ob-next" />
        </View>
      ) : null}

      {step === "location" ? (
        <View style={styles.step}>
          <Text variant="title">Où êtes-vous ?</Text>
          <Text tone="muted">
            Nous utilisons votre position pour proposer des personnes proches. Elle est arrondie à environ 1 km avant
            d'être enregistrée, et les autres membres ne voient jamais qu'une distance approximative.
          </Text>
          <View style={[styles.locationBox, { backgroundColor: colors.surface, borderColor: colors.border }]}>
            <Text variant="heading">
              {position || profile?.hasLocation
                ? city
                  ? `📍 ${city}`
                  : "📍 Position enregistrée"
                : "Position non définie"}
            </Text>
          </View>
          <Input label="Ville (facultatif)" maxLength={100} onChangeText={setCity} testID="ob-city" value={city} />
          <Button
            label={position ? "Actualiser ma position" : "Utiliser ma position"}
            loading={busy}
            onPress={locate}
            testID="ob-locate"
            variant="secondary"
          />
          <Button
            disabled={!position && !profile?.hasLocation}
            label="Continuer"
            loading={busy}
            onPress={onLocation}
            testID="ob-next"
          />
        </View>
      ) : null}

      {step === "preferences" ? (
        <View style={styles.step}>
          <Text variant="title">Que recherchez-vous ?</Text>
          <PreferencesForm onChange={setPrefs} value={prefs} />
          <Button label="Continuer" loading={busy} onPress={onPreferences} testID="ob-next" />
        </View>
      ) : null}

      {step === "photos" ? (
        <View style={styles.step}>
          <Text variant="title">Vos photos</Text>
          <Text tone="muted">Ajoutez de 1 à 6 photos. La première sera votre photo principale.</Text>
          <PhotoGrid />
          <Button label="Continuer" loading={busy} onPress={onPhotos} testID="ob-next" />
        </View>
      ) : null}

      {step === "about" ? (
        <View style={styles.step}>
          <Text variant="title">Un peu plus sur vous</Text>
          <Input
            label="Présentation"
            maxLength={500}
            multiline
            onChangeText={setBio}
            placeholder="Ce que vous aimez, ce que vous cherchez…"
            testID="ob-bio"
            value={bio}
          />
          <Text variant="heading">Centres d'intérêt</Text>
          <InterestPicker onChange={setInterests} selected={interests} />
          <Button label="Terminer" loading={busy} onPress={onAbout} testID="ob-finish" />
        </View>
      ) : null}

      {error ? (
        <Text accessibilityRole="alert" style={[styles.error, { backgroundColor: colors.dangerSoft }]} tone="danger">
          {error}
        </Text>
      ) : null}

      <View style={styles.footer}>
        {index > 0 && step !== "photos" && step !== "about" ? (
          <Button label="Retour" onPress={() => setStep(ORDER[index - 1])} small variant="ghost" />
        ) : null}
        {index === 0 ? <Button label="Me déconnecter" onPress={() => void logout()} small variant="ghost" /> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.sm, marginVertical: spacing.lg },
  dot: { flex: 1, height: 5, borderRadius: 3 },
  step: { gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  locationBox: { padding: spacing.lg, borderRadius: radii.md, borderWidth: 1 },
  error: { padding: spacing.md, borderRadius: radii.md, overflow: "hidden", marginTop: spacing.lg },
  footer: { flexDirection: "row", marginTop: spacing.md }
});
