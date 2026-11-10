import React, { useMemo, useRef, useState } from "react";
import { StyleSheet, TextInput, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, MeProfile, Photo, RelationshipGoal } from "../../api/types";
import { keys } from "../../hooks/queries";
import { useLocationSync } from "../../hooks/useLocation";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Chip, IconButton, Screen, Text, TextField, showToast } from "../../ui";
import { genderLabels, goalLabels, interestedInLabels } from "../../utils/format";
import { isValidFirstName, parseBirthdate } from "../../utils/validation";
import { InterestPicker } from "../profile/InterestPicker";
import { PhotoGrid } from "../profile/PhotoGrid";

type Step = "name" | "birthdate" | "gender" | "looking" | "photos" | "about" | "location";
const STEPS: Step[] = ["name", "birthdate", "gender", "looking", "photos", "about", "location"];

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];
const GOALS: RelationshipGoal[] = ["long_term", "short_term", "friendship", "undecided"];

/** First step not yet satisfied by the saved profile (resume onboarding). */
function initialStep(me: MeProfile): Step {
  if (!me.profile) return "name";
  if (!me.preferences) return "looking";
  if (me.photos.length === 0) return "photos";
  return "about";
}

export function OnboardingScreen({ me }: { me: MeProfile }) {
  const theme = useTheme();
  const qc = useQueryClient();
  const syncLocation = useLocationSync();
  const [step, setStep] = useState<Step>(() => initialStep(me));
  const [firstName, setFirstName] = useState(me.profile?.firstName ?? "");
  const [day, setDay] = useState(me.profile?.birthdate.slice(8, 10) ?? "");
  const [month, setMonth] = useState(me.profile?.birthdate.slice(5, 7) ?? "");
  const [year, setYear] = useState(me.profile?.birthdate.slice(0, 4) ?? "");
  const [gender, setGender] = useState<Gender | null>(me.profile?.gender ?? null);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(me.preferences?.interestedIn ?? []);
  const [goal, setGoal] = useState<RelationshipGoal | null>(me.profile?.relationshipGoal ?? null);
  const [photos, setPhotos] = useState<Photo[]>(me.photos);
  const [bio, setBio] = useState(me.profile?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(me.profile?.jobTitle ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(me.interests.map((i) => i.id));
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);

  const index = STEPS.indexOf(step);
  const birth = useMemo(() => parseBirthdate(day, month, year), [day, month, year]);

  const saveProfile = async () => {
    if (!birth.iso || !gender) return;
    await profileApi.upsert({
      firstName: firstName.trim(),
      birthdate: birth.iso,
      gender,
      bio: bio.trim(),
      jobTitle: jobTitle.trim(),
      city: me.profile?.city ?? "",
      heightCm: me.profile?.heightCm ?? null,
      relationshipGoal: goal,
      interestIds
    });
  };

  const finish = async () => {
    await qc.invalidateQueries({ queryKey: keys.me });
  };

  const next = async () => {
    setError(null);
    try {
      switch (step) {
        case "name":
          if (!isValidFirstName(firstName)) return setError("Votre prénom, uniquement avec des lettres (30 max).");
          break;
        case "birthdate":
          if (birth.error) return setError(birth.error);
          break;
        case "gender":
          if (!gender) return setError("Choisissez une option.");
          break;
        case "looking":
          if (interestedIn.length === 0) return setError("Choisissez au moins une option.");
          setSaving(true);
          await saveProfile();
          await profileApi.updatePreferences({
            interestedIn,
            minAge: me.preferences?.minAge ?? 18,
            maxAge: me.preferences?.maxAge ?? 99,
            maxDistanceKm: me.preferences?.maxDistanceKm ?? 50
          });
          break;
        case "photos":
          if (photos.length === 0) return setError("Ajoutez au moins une photo de vous.");
          break;
        case "about":
          setSaving(true);
          await saveProfile();
          break;
        case "location": {
          setSaving(true);
          const outcome = await syncLocation();
          if (outcome !== "updated") {
            showToast("Localisation indisponible : vous pourrez l'activer plus tard depuis Découvrir.", "info");
          }
          await finish();
          return;
        }
      }
      setStep(STEPS[index + 1]);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  const toggleInterested = (g: Gender) => setInterestedIn((v) => (v.includes(g) ? v.filter((x) => x !== g) : [...v, g]));

  return (
    <Screen
      scroll
      edges={["top", "bottom", "left", "right"]}
      footer={
        <View style={styles.footer}>
          {error ? (
            <Text accessibilityLiveRegion="polite" tone="danger" variant="small">
              {error}
            </Text>
          ) : null}
          <Button label={step === "location" ? "Activer la localisation" : "Continuer"} loading={saving} onPress={next} testID="onboarding-next" />
          {step === "location" ? <Button label="Plus tard" onPress={finish} variant="ghost" testID="onboarding-skip-location" /> : null}
        </View>
      }
    >
      <View style={styles.top}>
        {index > 0 && step !== "photos" ? (
          <IconButton icon="chevron-back" label="Étape précédente" onPress={() => setStep(STEPS[index - 1])} />
        ) : (
          <View style={styles.backPlaceholder} />
        )}
        <View style={[styles.progress, { backgroundColor: theme.colors.surfaceAlt }]}>
          <View style={[styles.progressFill, { width: `${((index + 1) / STEPS.length) * 100}%`, backgroundColor: theme.colors.primary }]} />
        </View>
      </View>

      <View style={styles.body}>
        {step === "name" ? (
          <>
            <Text variant="title">Comment vous appelez-vous ?</Text>
            <Text tone="muted">Seul votre prénom est affiché sur votre profil.</Text>
            <TextField
              autoCapitalize="words"
              autoComplete="given-name"
              autoFocus
              label="Prénom"
              maxLength={30}
              onChangeText={setFirstName}
              onSubmitEditing={next}
              testID="onboarding-firstname"
              value={firstName}
            />
          </>
        ) : null}

        {step === "birthdate" ? (
          <>
            <Text variant="title">Votre date de naissance</Text>
            <Text tone="muted">Seul votre âge sera visible. Elle ne pourra plus être modifiée ensuite.</Text>
            <View style={styles.dateRow}>
              <View style={styles.dateCell}>
                <TextField
                  inputMode="numeric"
                  keyboardType="number-pad"
                  label="Jour"
                  maxLength={2}
                  onChangeText={(v) => {
                    setDay(v.replace(/\D/g, ""));
                    if (v.length === 2) monthRef.current?.focus();
                  }}
                  placeholder="JJ"
                  testID="onboarding-day"
                  value={day}
                />
              </View>
              <View style={styles.dateCell}>
                <TextField
                  ref={monthRef}
                  inputMode="numeric"
                  keyboardType="number-pad"
                  label="Mois"
                  maxLength={2}
                  onChangeText={(v) => {
                    setMonth(v.replace(/\D/g, ""));
                    if (v.length === 2) yearRef.current?.focus();
                  }}
                  placeholder="MM"
                  testID="onboarding-month"
                  value={month}
                />
              </View>
              <View style={styles.yearCell}>
                <TextField
                  ref={yearRef}
                  inputMode="numeric"
                  keyboardType="number-pad"
                  label="Année"
                  maxLength={4}
                  onChangeText={(v) => setYear(v.replace(/\D/g, ""))}
                  placeholder="AAAA"
                  testID="onboarding-year"
                  value={year}
                />
              </View>
            </View>
          </>
        ) : null}

        {step === "gender" ? (
          <>
            <Text variant="title">Vous êtes…</Text>
            <View style={styles.options}>
              {GENDERS.map((g) => (
                <Chip key={g} label={genderLabels[g]} onPress={() => setGender(g)} selected={gender === g} testID={`gender-${g}`} />
              ))}
            </View>
          </>
        ) : null}

        {step === "looking" ? (
          <>
            <Text variant="title">Vous souhaitez rencontrer…</Text>
            <View style={styles.options}>
              {GENDERS.map((g) => (
                <Chip
                  key={g}
                  label={interestedInLabels[g]}
                  onPress={() => toggleInterested(g)}
                  selected={interestedIn.includes(g)}
                  testID={`interested-${g}`}
                />
              ))}
            </View>
            <Text style={styles.subTitle} variant="heading">
              Ce que vous cherchez{" "}
              <Text tone="subtle" variant="small">
                (facultatif)
              </Text>
            </Text>
            <View style={styles.options}>
              {GOALS.map((g) => (
                <Chip key={g} label={goalLabels[g]} onPress={() => setGoal(goal === g ? null : g)} selected={goal === g} />
              ))}
            </View>
          </>
        ) : null}

        {step === "photos" ? (
          <>
            <Text variant="title">Ajoutez vos photos</Text>
            <Text tone="muted">Au moins une photo où l’on voit bien votre visage. Pas de photos d’autres personnes sans leur accord.</Text>
            <PhotoGrid onChange={setPhotos} photos={photos} />
          </>
        ) : null}

        {step === "about" ? (
          <>
            <Text variant="title">Parlez un peu de vous</Text>
            <TextField
              counter
              label="Bio"
              maxLength={500}
              multiline
              onChangeText={setBio}
              placeholder="Ce qui vous fait sourire, ce que vous aimez partager…"
              testID="onboarding-bio"
              value={bio}
            />
            <TextField label="Métier (facultatif)" maxLength={60} onChangeText={setJobTitle} value={jobTitle} />
            <Text style={styles.subTitle} variant="heading">
              Centres d’intérêt
            </Text>
            <InterestPicker onChange={setInterestIds} value={interestIds} />
          </>
        ) : null}

        {step === "location" ? (
          <>
            <Text variant="title">Trouver des personnes près de vous</Text>
            <Text tone="muted">
              Lueur utilise votre position uniquement pour calculer une distance approximative. Votre position exacte n’est jamais enregistrée ni montrée : nous
              ne gardons qu’une zone d’environ 1 km.
            </Text>
          </>
        ) : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  top: { flexDirection: "row", alignItems: "center", gap: 8, paddingTop: 8 },
  backPlaceholder: { width: 44, height: 44 },
  progress: { flex: 1, height: 6, borderRadius: 3, overflow: "hidden" },
  progressFill: { height: 6, borderRadius: 3 },
  body: { gap: 16, paddingTop: 24 },
  dateRow: { flexDirection: "row", gap: 12 },
  dateCell: { flex: 1 },
  yearCell: { flex: 1.5 },
  options: { flexDirection: "row", flexWrap: "wrap", gap: 10 },
  subTitle: { marginTop: 8 },
  footer: { gap: 10 }
});
