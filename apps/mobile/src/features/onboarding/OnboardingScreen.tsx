import React, { useEffect, useMemo, useState } from "react";
import { View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, RelationshipGoal } from "../../api/types";
import { useOnboardingGate } from "../../navigation/onboardingGate";
import { makeStyles } from "../../theme";
import { Button, Chip, Icon, LoadingState, Screen, Section, showToast, Text, TextField } from "../../ui";
import { ageFromBirthdate, genderLabels, goalLabels, seekingLabels } from "../../utils/format";
import { useAuth } from "../auth/AuthProvider";
import { BirthdateField } from "../profile/BirthdateField";
import { useMyProfile, useSetMyProfile } from "../profile/hooks";
import { InterestPicker } from "../profile/InterestPicker";
import { shareApproximateLocation } from "../profile/location";
import { PhotoGrid } from "../profile/PhotoGrid";

const STEPS = ["identity", "seeking", "photos", "about", "location"] as const;
type Step = (typeof STEPS)[number];
const genders: Gender[] = ["woman", "man", "nonbinary"];
const goals = Object.keys(goalLabels) as RelationshipGoal[];

export function OnboardingScreen() {
  const styles = useStyles();
  const { signOut } = useAuth();
  const gate = useOnboardingGate();
  const queryClient = useQueryClient();
  const setProfile = useSetMyProfile();
  const profileQuery = useMyProfile();
  const me = profileQuery.data;

  const [step, setStep] = useState<Step | null>(null);
  const [firstName, setFirstName] = useState("");
  const [birthdate, setBirthdate] = useState("");
  const [gender, setGender] = useState<Gender | null>(null);
  const [interestedIn, setInterestedIn] = useState<Gender[]>([]);
  const [bio, setBio] = useState("");
  const [goal, setGoal] = useState<RelationshipGoal | null>(null);
  const [interestIds, setInterestIds] = useState<number[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Resume where the member left off.
  useEffect(() => {
    if (!me || step !== null) return;
    if (me.profile) {
      setFirstName(me.profile.firstName);
      setBirthdate(me.profile.birthdate);
      setGender(me.profile.gender);
      setBio(me.profile.bio);
      setGoal(me.profile.relationshipGoal);
      setInterestedIn(me.preferences?.interestedIn ?? []);
      setInterestIds(me.interests.map((i) => i.id));
      setStep(me.photos.length === 0 ? "photos" : "about");
    } else {
      setStep("identity");
    }
  }, [me, step]);

  const index = step ? STEPS.indexOf(step) : 0;
  const age = useMemo(() => (birthdate ? ageFromBirthdate(birthdate) : null), [birthdate]);

  const go = (next: Step) => {
    setError(null);
    setStep(next);
  };

  const saveProfile = async (extra?: { bio?: string; relationshipGoal?: RelationshipGoal | null }) => {
    const current = me?.profile;
    const updated = await profileApi.upsert({
      firstName: firstName.trim(),
      birthdate,
      gender: gender as Gender,
      bio: extra?.bio ?? current?.bio ?? "",
      city: current?.city ?? "",
      jobTitle: current?.jobTitle ?? "",
      heightCm: current?.heightCm ?? null,
      relationshipGoal: extra?.relationshipGoal !== undefined ? extra.relationshipGoal : (current?.relationshipGoal ?? null)
    });
    setProfile(updated);
  };

  const next = async () => {
    if (!step) return;
    setError(null);
    if (step === "identity") {
      if (firstName.trim().length < 1) return setError("Indiquez votre prénom.");
      if (age === null) return setError("Date de naissance invalide.");
      if (age < 18) return setError("Lueur est réservé aux personnes majeures (18 ans et plus).");
      if (age > 99) return setError("Date de naissance invalide.");
      return go("seeking");
    }
    if (step === "seeking") {
      if (!gender) return setError("Choisissez comment vous vous identifiez.");
      if (interestedIn.length === 0) return setError("Choisissez qui vous souhaitez rencontrer.");
      setSaving(true);
      try {
        await saveProfile();
        const prefs = me?.preferences;
        setProfile(
          await profileApi.preferences({
            interestedIn,
            minAge: prefs?.minAge ?? Math.max(18, (age ?? 30) - 8),
            maxAge: prefs?.maxAge ?? Math.min(99, (age ?? 30) + 8),
            maxDistanceKm: prefs?.maxDistanceKm ?? 50
          })
        );
        go("photos");
      } catch (err) {
        setError(errorMessage(err));
      } finally {
        setSaving(false);
      }
      return;
    }
    if (step === "photos") {
      if ((me?.photos.length ?? 0) === 0) return setError("Ajoutez au moins une photo.");
      return go("about");
    }
    if (step === "about") {
      setSaving(true);
      try {
        await saveProfile({ bio, relationshipGoal: goal });
        setProfile(await profileApi.interests(interestIds));
        go("location");
      } catch (err) {
        setError(errorMessage(err));
      } finally {
        setSaving(false);
      }
      return;
    }
    if (step === "location") {
      await finish();
    }
  };

  const finish = async () => {
    await queryClient.invalidateQueries({ queryKey: queryKeys.myProfile });
    gate.finish();
  };

  const shareLocation = async () => {
    setSaving(true);
    try {
      const result = await shareApproximateLocation();
      if (!result.ok) {
        showToast(
          result.reason === "denied" ? "Localisation refusée : vous pourrez l'activer plus tard." : "Position indisponible pour le moment.",
          "error"
        );
        return;
      }
      if (result.city && me?.profile && !me.profile.city) {
        await profileApi.upsert({ ...me.profile, city: result.city });
      }
      showToast("Position approximative enregistrée.", "success");
      await finish();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setSaving(false);
    }
  };

  if (!me || !step) {
    return <LoadingState label="Préparation de votre profil…" />;
  }

  const back = index > 0 && step !== "photos" ? () => go(STEPS[index - 1]) : step === "photos" && !me.profile ? () => go("seeking") : null;

  return (
    <Screen
      scroll
      keyboard
      footer={
        <>
          {error ? (
            <Text variant="caption" tone="danger" align="center" accessibilityLiveRegion="polite">
              {error}
            </Text>
          ) : null}
          {step === "location" ? (
            <>
              <Button
                testID="onboarding-location"
                label="Partager ma position approximative"
                icon="navigate-outline"
                size="lg"
                loading={saving}
                onPress={shareLocation}
              />
              <Button testID="onboarding-skip-location" label="Plus tard" variant="ghost" onPress={() => void finish()} />
            </>
          ) : (
            <Button testID="onboarding-next" label="Continuer" size="lg" loading={saving} onPress={() => void next()} />
          )}
        </>
      }
    >
      <View style={styles.topBar}>
        {back ? <Button label="Retour" variant="ghost" size="sm" icon="chevron-back" onPress={back} /> : <View />}
        <Button label="Se déconnecter" variant="ghost" size="sm" onPress={() => void signOut()} />
      </View>
      <View style={styles.progress} accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.dot, i <= index ? styles.dotActive : null]} />
        ))}
      </View>

      {step === "identity" ? (
        <>
          <Text variant="title" accessibilityRole="header">
            Faisons connaissance
          </Text>
          <TextField
            testID="onboarding-firstname"
            label="Prénom"
            value={firstName}
            onChangeText={setFirstName}
            autoComplete="given-name"
            maxLength={40}
            placeholder="Votre prénom"
          />
          <BirthdateField value={birthdate} onChange={setBirthdate} />
        </>
      ) : null}

      {step === "seeking" ? (
        <>
          <Text variant="title" accessibilityRole="header">
            Vous et vos envies
          </Text>
          <Section title="Je suis">
            <View style={styles.chips}>
              {genders.map((g) => (
                <Chip key={g} testID={`gender-${g}`} label={genderLabels[g]} selected={gender === g} onPress={() => setGender(g)} />
              ))}
            </View>
          </Section>
          <Section
            title="Je souhaite rencontrer"
            hint="Plusieurs choix possibles. Vous ne verrez que les personnes qui souhaitent aussi vous rencontrer."
          >
            <View style={styles.chips}>
              {genders.map((g) => (
                <Chip
                  key={g}
                  testID={`seeking-${g}`}
                  label={seekingLabels[g]}
                  selected={interestedIn.includes(g)}
                  onPress={() => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
                />
              ))}
            </View>
          </Section>
        </>
      ) : null}

      {step === "photos" ? (
        <>
          <Text variant="title" accessibilityRole="header">
            Vos plus belles photos
          </Text>
          <Text tone="muted">
            Ajoutez de 1 à 6 photos. La première sera votre photo principale. Les métadonnées (dont la position GPS) sont supprimées.
          </Text>
          <PhotoGrid photos={me.photos} minimum={0} />
        </>
      ) : null}

      {step === "about" ? (
        <>
          <Text variant="title" accessibilityRole="header">
            Ce qui vous ressemble
          </Text>
          <TextField
            testID="onboarding-bio"
            label="Bio"
            value={bio}
            onChangeText={setBio}
            multiline
            counter={500}
            placeholder="Quelques mots sur vous, ce que vous aimez, ce que vous cherchez…"
          />
          <Section title="Je recherche">
            <View style={styles.chips}>
              {goals.map((g) => (
                <Chip key={g} label={goalLabels[g]} selected={goal === g} onPress={() => setGoal(goal === g ? null : g)} />
              ))}
            </View>
          </Section>
          <Section title="Centres d'intérêt">
            <InterestPicker selected={interestIds} onChange={setInterestIds} />
          </Section>
        </>
      ) : null}

      {step === "location" ? (
        <View style={styles.locationIntro}>
          <View style={styles.locationIcon}>
            <Icon name="location-outline" size={34} color="primary" />
          </View>
          <Text variant="title" align="center" accessibilityRole="header">
            Des rencontres près de vous
          </Text>
          <Text tone="muted" align="center">
            Votre position sert uniquement à calculer une distance approximative. Elle est arrondie à environ 1 km, jamais affichée, et vous
            pouvez la supprimer à tout moment.
          </Text>
        </View>
      ) : null}
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  topBar: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", marginHorizontal: -t.spacing.md },
  progress: { flexDirection: "row", gap: t.spacing.xs },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: t.colors.border },
  dotActive: { backgroundColor: t.colors.primary },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm },
  locationIntro: { alignItems: "center", gap: t.spacing.md, paddingTop: t.spacing.xxl },
  locationIcon: {
    width: 76,
    height: 76,
    borderRadius: 38,
    backgroundColor: t.colors.primarySoft,
    alignItems: "center",
    justifyContent: "center"
  }
}));
