import React, { useEffect, useMemo, useState } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, OwnProfile, Preferences, RelationshipGoal } from "../../api/types";
import { InterestPicker } from "../../components/InterestPicker";
import { getCoarseLocation } from "../../components/location";
import { PhotoGrid } from "../../components/PhotoGrid";
import { PreferencesForm } from "../../components/PreferencesForm";
import { useAuth } from "../../hooks/useAuth";
import { useInterests, useSetProfile } from "../../hooks/queries";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Chip, IconButton, InlineError, Loading, ProgressBar, Screen, Text, TextField, showToast } from "../../ui";
import { genderLabels, goalLabels, parseBirthdate } from "../../utils/format";

const STEPS = ["basics", "preferences", "about", "location", "photos"] as const;
type Step = (typeof STEPS)[number];

function initialStep(profile: OwnProfile | null): Step {
  if (!profile) return "basics";
  if (profile.missing.includes("preferences")) return "preferences";
  return "photos";
}

const defaultPrefs: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

export default function OnboardingScreen({ profile }: { profile: OwnProfile | null }) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const setProfile = useSetProfile();
  const interests = useInterests();
  const [step, setStep] = useState<Step>(() => initialStep(profile));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Basics
  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [day, setDay] = useState(profile?.birthdate.slice(8, 10) ?? "");
  const [month, setMonth] = useState(profile?.birthdate.slice(5, 7) ?? "");
  const [year, setYear] = useState(profile?.birthdate.slice(0, 4) ?? "");
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  // Preferences
  const [prefs, setPrefs] = useState<Preferences>(profile && profile.preferences.interestedIn.length ? profile.preferences : defaultPrefs);
  const [goal, setGoal] = useState<RelationshipGoal>(profile?.relationshipGoal ?? "unsure");
  // About
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [occupation, setOccupation] = useState(profile?.occupation ?? "");
  const [selected, setSelected] = useState<number[]>(profile?.interests.map((i) => i.id) ?? []);
  // Location
  const [city, setCity] = useState(profile?.city ?? "");

  const index = STEPS.indexOf(step);
  useEffect(() => setError(null), [step]);

  const save = async (action: () => Promise<OwnProfile>, next?: Step) => {
    setBusy(true);
    setError(null);
    try {
      const updated = await action();
      if (next) setStep(next);
      setProfile(updated);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const profileInput = () => {
    const birth = parseBirthdate(day, month, year);
    return {
      birth,
      input: { firstName: firstName.trim(), birthdate: birth.iso ?? "", gender: gender ?? "woman", bio, occupation, relationshipGoal: goal, city }
    };
  };

  const submitBasics = () => {
    const { birth, input } = profileInput();
    if (!firstName.trim()) return setError("Indiquez votre prénom.");
    if (birth.error) return setError(birth.error);
    if (!gender) return setError("Choisissez votre genre.");
    void save(() => profileApi.upsert(input), "preferences");
  };

  const submitPreferences = () => {
    if (prefs.interestedIn.length === 0) return setError("Choisissez au moins une option.");
    const { input } = profileInput();
    void save(async () => {
      await profileApi.upsert(input);
      return profileApi.preferences(prefs);
    }, "about");
  };

  const submitAbout = () => {
    const { input } = profileInput();
    void save(async () => {
      await profileApi.upsert(input);
      return profileApi.interests(selected);
    }, "location");
  };

  const shareLocation = async () => {
    setBusy(true);
    const loc = await getCoarseLocation();
    setBusy(false);
    if ("error" in loc) {
      setError(loc.error);
      return;
    }
    if (loc.city && !city) setCity(loc.city);
    void save(() => profileApi.location(loc.latitude, loc.longitude, loc.city ?? (city || undefined)), "photos");
    showToast("Position approximative enregistrée", "success");
  };

  const skipLocation = () => {
    const { input } = profileInput();
    void save(() => profileApi.upsert(input), "photos");
  };

  const canFinish = (profile?.photos.length ?? 0) > 0;
  const titles: Record<Step, [string, string]> = useMemo(
    () => ({
      basics: ["Faisons connaissance", "Ces informations apparaîtront sur votre profil. Votre date de naissance reste privée : seul votre âge est affiché."],
      preferences: ["Qui aimeriez-vous rencontrer ?", "Vous pourrez changer cela à tout moment."],
      about: ["Parlez un peu de vous", "Facultatif, mais les profils complets reçoivent bien plus de messages."],
      location: ["Près de chez vous", "Nous n'affichons qu'une distance approximative, jamais votre position exacte."],
      photos: ["Vos photos", "Ajoutez au moins une photo où l'on voit bien votre visage. 6 maximum."]
    }),
    []
  );

  if (!interests.data && interests.isLoading) {
    return <Loading />;
  }

  return (
    <Screen scroll keyboard>
      <View style={styles.top}>
        {index > 0 ? (
          <IconButton icon="chevron-back" label="Étape précédente" onPress={() => setStep(STEPS[index - 1] as Step)} />
        ) : (
          <View style={styles.spacer} />
        )}
        <View style={styles.progress}>
          <ProgressBar value={(index + 1) / STEPS.length} />
        </View>
        <Button label="Quitter" variant="ghost" compact onPress={() => void signOut()} />
      </View>

      <View style={styles.head}>
        <Text variant="overline" tone="primary">
          Étape {index + 1} sur {STEPS.length}
        </Text>
        <Text variant="title" serif accessibilityRole="header">
          {titles[step][0]}
        </Text>
        <Text tone="muted">{titles[step][1]}</Text>
      </View>

      <View style={styles.body}>
        {step === "basics" ? (
          <>
            <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} autoComplete="given-name" testID="onboarding-firstname" />
            <View style={styles.gap}>
              <Text variant="label" tone="muted">
                Date de naissance
              </Text>
              <View style={styles.dateRow}>
                {[
                  { v: day, set: setDay, ph: "JJ", len: 2, id: "day" },
                  { v: month, set: setMonth, ph: "MM", len: 2, id: "month" },
                  { v: year, set: setYear, ph: "AAAA", len: 4, id: "year" }
                ].map((f) => (
                  <TextInput
                    key={f.id}
                    accessibilityLabel={f.id === "day" ? "Jour" : f.id === "month" ? "Mois" : "Année"}
                    value={f.v}
                    onChangeText={(t) => f.set(t.replace(/\D/g, "").slice(0, f.len))}
                    placeholder={f.ph}
                    placeholderTextColor={colors.textSubtle}
                    keyboardType="number-pad"
                    editable={!profile}
                    testID={`onboarding-birth-${f.id}`}
                    style={[styles.dateInput, { flex: f.len === 4 ? 1.6 : 1, color: colors.text, borderColor: colors.border, backgroundColor: colors.surface }]}
                  />
                ))}
              </View>
              {profile ? (
                <Text variant="caption" tone="subtle">
                  La date de naissance ne peut plus être modifiée.
                </Text>
              ) : null}
            </View>
            <View style={styles.gap}>
              <Text variant="label" tone="muted">
                Je suis
              </Text>
              <View style={styles.chips}>
                {(Object.keys(genderLabels) as Gender[]).map((g) => (
                  <Chip key={g} label={genderLabels[g]} selected={gender === g} onPress={() => setGender(g)} />
                ))}
              </View>
            </View>
            <InlineError message={error} />
            <Button label="Continuer" onPress={submitBasics} loading={busy} testID="onboarding-next" />
          </>
        ) : null}

        {step === "preferences" ? (
          <>
            <PreferencesForm value={prefs} onChange={setPrefs} />
            <View style={styles.gap}>
              <Text variant="label" tone="muted">
                Je cherche plutôt
              </Text>
              <View style={styles.chips}>
                {(Object.keys(goalLabels) as RelationshipGoal[]).map((g) => (
                  <Chip key={g} label={goalLabels[g]} selected={goal === g} onPress={() => setGoal(g)} />
                ))}
              </View>
            </View>
            <InlineError message={error} />
            <Button label="Continuer" onPress={submitPreferences} loading={busy} testID="onboarding-next" />
          </>
        ) : null}

        {step === "about" ? (
          <>
            <TextField
              label="Bio"
              value={bio}
              onChangeText={setBio}
              multiline
              maxLength={500}
              counter
              placeholder="Ce qui vous fait vibrer, votre dimanche idéal…"
            />
            <TextField label="Métier (facultatif)" value={occupation} onChangeText={setOccupation} maxLength={80} />
            <InterestPicker
              catalogue={interests.data ?? []}
              selected={selected}
              onToggle={(id) => setSelected((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]))}
            />
            <InlineError message={error} />
            <Button label="Continuer" onPress={submitAbout} loading={busy} testID="onboarding-next" />
          </>
        ) : null}

        {step === "location" ? (
          <>
            <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>
              <Text variant="bodyStrong">Pourquoi la localisation ?</Text>
              <Text tone="muted">
                Pour vous proposer des personnes proches. Votre position est arrondie à environ 1 km, uniquement mise à jour quand vous le décidez, et jamais
                montrée aux autres.
              </Text>
            </View>
            <Button label="Utiliser ma position approximative" icon="navigate" onPress={shareLocation} loading={busy} testID="onboarding-location" />
            <TextField label="Ou indiquez votre ville" value={city} onChangeText={setCity} maxLength={80} autoComplete="postal-address-locality" />
            <InlineError message={error} />
            <Button label={city ? "Continuer avec cette ville" : "Plus tard"} variant="secondary" onPress={skipLocation} testID="onboarding-skip-location" />
          </>
        ) : null}

        {step === "photos" ? (
          <>
            <PhotoGrid photos={profile?.photos ?? []} onChange={(photos) => profile && setProfile({ ...profile, photos })} />
            <InlineError message={error} />
            <Button
              label="Découvrir des profils"
              disabled={!canFinish}
              onPress={() => void save(() => profileApi.get())}
              loading={busy}
              testID="onboarding-finish"
            />
            {!canFinish ? (
              <Text variant="caption" tone="subtle" center>
                Une photo minimum pour continuer.
              </Text>
            ) : null}
          </>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { flexDirection: "row", alignItems: "center", gap: spacing.xs, paddingTop: spacing.xs },
  spacer: { width: 44 },
  progress: { flex: 1 },
  head: { gap: spacing.xs, marginTop: spacing.lg, marginBottom: spacing.lg },
  body: { gap: spacing.md },
  gap: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  dateRow: { flexDirection: "row", gap: spacing.xs },
  dateInput: { borderWidth: 1.5, borderRadius: radii.md, minHeight: 52, textAlign: "center", fontSize: 17 },
  card: { borderWidth: 1, borderRadius: radii.lg, padding: spacing.md, gap: spacing.xs }
});
