import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { Gender, LookingFor, OwnProfile, ProfileInput } from "../../api/types";
import { genderLabels, interestedInLabels, lookingForLabels } from "../../lib/format";
import { getApproximateLocation } from "../../lib/location";
import { ageFrom, birthdateProblem, hasLetter, isoToFrenchDate, maskDateInput, parseFrenchDate } from "../../lib/validation";
import { useSession, useSetProfile } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { Button, Header, Icon, Screen, Text, TextField, useFeedback } from "../../ui";
import { ChoiceList } from "../profile/ChoiceList";
import { InterestPicker } from "../profile/InterestPicker";
import { PhotoGrid } from "../profile/PhotoGrid";

const STEPS = ["Vous", "Rencontres", "Photos", "À propos", "Localisation"] as const;

type Props = { onStart: () => void; onFinish: () => void };

function initialStep(profile: OwnProfile | null | undefined): number {
  if (!profile) return 0;
  if (profile.photos.length === 0) return 2;
  return 3;
}

export function OnboardingScreen({ onStart, onFinish }: Props) {
  const { colors } = useTheme();
  const { profile, signOut } = useSession();
  const setProfile = useSetProfile();
  const { toast } = useFeedback();
  const [step, setStep] = useState(() => initialStep(profile));
  const [submitted, setSubmitted] = useState(false);

  const [firstName, setFirstName] = useState(profile?.firstName ?? "");
  const [birthdate, setBirthdate] = useState(profile ? isoToFrenchDate(profile.birthdate) : "");
  const [gender, setGender] = useState<Gender | null>(profile?.gender ?? null);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile?.preferences.interestedIn ?? []);
  const [lookingFor, setLookingFor] = useState<LookingFor>(profile?.lookingFor ?? "unsure");
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(profile?.jobTitle ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(profile?.interests.map((i) => i.id) ?? []);
  const [city, setCity] = useState(profile?.city ?? "");

  useEffect(() => {
    onStart();
  }, [onStart]);

  const input = (): ProfileInput => ({
    firstName: firstName.trim(),
    birthdate: parseFrenchDate(birthdate) ?? "",
    gender: gender ?? "woman",
    bio,
    jobTitle,
    lookingFor,
    city,
    interestIds
  });

  const saveBasics = useMutation({
    mutationFn: async () => {
      const saved = await profileApi.save(input());
      const iso = parseFrenchDate(birthdate) ?? saved.birthdate;
      const [y, m, d] = iso.split("-").map(Number);
      const age = ageFrom(new Date(y, m - 1, d));
      const preferences = {
        interestedIn,
        ageMin: Math.max(18, age - 8),
        ageMax: Math.min(99, age + 8),
        maxDistanceKm: 50,
        isDiscoverable: true,
        showDistance: true
      };
      await profileApi.savePreferences(preferences);
      return { ...saved, preferences };
    },
    onSuccess: (p) => {
      setProfile(p);
      next();
    }
  });

  const saveAbout = useMutation({
    mutationFn: () => profileApi.save(input()),
    onSuccess: (p) => {
      setProfile(p);
      next();
    }
  });

  const locate = useMutation({
    mutationFn: async () => {
      const loc = await getApproximateLocation();
      await profileApi.saveLocation(loc.latitude, loc.longitude, loc.city);
      if (loc.city) setCity(loc.city);
      return profileApi.get();
    },
    onSuccess: (p) => {
      setProfile(p);
      toast("Position approximative enregistrée", "success");
      onFinish();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const finishWithoutLocation = useMutation({
    mutationFn: () => profileApi.save(input()),
    onSuccess: (p) => {
      setProfile(p);
      onFinish();
    }
  });

  function next() {
    setSubmitted(false);
    setStep((s) => Math.min(STEPS.length - 1, s + 1));
  }

  const nameError = submitted && !hasLetter(firstName) ? "Indiquez votre prénom." : null;
  const dateError = submitted ? birthdateProblem(birthdate) : null;

  let content: React.ReactNode;
  let primary: { label: string; onPress: () => void; loading?: boolean; disabled?: boolean };
  const pendingError = saveBasics.error ?? saveAbout.error ?? finishWithoutLocation.error;

  switch (step) {
    case 0:
      content = (
        <>
          <Text variant="title">Faisons connaissance</Text>
          <Text tone="muted">Votre prénom et votre âge seront visibles. Votre date de naissance exacte, jamais.</Text>
          <TextField
            label="Prénom"
            value={firstName}
            onChangeText={setFirstName}
            maxLength={40}
            autoComplete="given-name"
            error={nameError}
            testID="onb-firstname"
          />
          <TextField
            label="Date de naissance"
            placeholder="JJ/MM/AAAA"
            value={birthdate}
            onChangeText={(v) => setBirthdate(maskDateInput(v))}
            keyboardType="number-pad"
            maxLength={10}
            error={dateError}
            hint={profile ? "La date de naissance ne peut plus être modifiée." : "Elle ne pourra plus être modifiée ensuite."}
            editable={!profile}
            testID="onb-birthdate"
          />
        </>
      );
      primary = {
        label: "Continuer",
        onPress: () => {
          setSubmitted(true);
          if (hasLetter(firstName) && !birthdateProblem(birthdate)) next();
        }
      };
      break;
    case 1:
      content = (
        <>
          <Text variant="title">Vous êtes…</Text>
          <ChoiceList
            options={(Object.keys(genderLabels) as Gender[]).map((g) => ({ value: g, label: genderLabels[g] }))}
            selected={gender ? [gender] : []}
            onToggle={setGender}
            testIDPrefix="onb-gender"
          />
          <Text variant="heading" style={styles.subTitle}>
            Vous souhaitez rencontrer…
          </Text>
          <ChoiceList
            multiple
            options={(Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }))}
            selected={interestedIn}
            onToggle={(g) => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]))}
            testIDPrefix="onb-interested"
          />
          <Text variant="heading" style={styles.subTitle}>
            Vous cherchez…
          </Text>
          <ChoiceList
            options={(Object.keys(lookingForLabels) as LookingFor[]).map((l) => ({ value: l, label: lookingForLabels[l] }))}
            selected={[lookingFor]}
            onToggle={setLookingFor}
          />
          {submitted && (!gender || interestedIn.length === 0) ? (
            <Text tone="danger">Choisissez votre genre et au moins une préférence.</Text>
          ) : null}
        </>
      );
      primary = {
        label: "Continuer",
        loading: saveBasics.isPending,
        onPress: () => {
          setSubmitted(true);
          if (gender && interestedIn.length > 0) saveBasics.mutate();
        }
      };
      break;
    case 2:
      content = (
        <>
          <Text variant="title">Vos photos</Text>
          <Text tone="muted">Ajoutez au moins une photo où l’on voit bien votre visage. La première sera votre photo principale.</Text>
          {profile ? <PhotoGrid profile={profile} /> : null}
        </>
      );
      primary = {
        label: "Continuer",
        disabled: !profile || profile.photos.length === 0,
        onPress: next
      };
      break;
    case 3:
      content = (
        <>
          <Text variant="title">Parlez de vous</Text>
          <TextField
            label="Bio"
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
            counter
            placeholder="Ce qui vous fait sourire, vos dimanches idéaux…"
            testID="onb-bio"
          />
          <TextField label="Métier (facultatif)" value={jobTitle} onChangeText={setJobTitle} maxLength={60} />
          <Text variant="heading" style={styles.subTitle}>
            Centres d’intérêt
          </Text>
          <InterestPicker selected={interestIds} onChange={setInterestIds} />
        </>
      );
      primary = { label: "Continuer", loading: saveAbout.isPending, onPress: () => saveAbout.mutate() };
      break;
    default:
      content = (
        <>
          <View style={[styles.locIcon, { backgroundColor: colors.accentSoft }]}>
            <Icon name="location-outline" size={34} color={colors.accent} />
          </View>
          <Text variant="title" align="center">
            Des rencontres près de vous
          </Text>
          <Text tone="muted" align="center">
            Nous utilisons une position approximative (à environ 1 km près), uniquement quand vous le demandez. Les autres ne voient qu’une
            distance arrondie, jamais votre adresse.
          </Text>
          <TextField label="Ville affichée (facultatif)" value={city} onChangeText={setCity} maxLength={80} />
        </>
      );
      primary = { label: "Utiliser ma position", loading: locate.isPending, onPress: () => locate.mutate() };
  }

  return (
    <Screen
      scroll
      footer={
        <View style={styles.footer}>
          {pendingError ? <Text tone="danger">{errorMessage(pendingError)}</Text> : null}
          <Button label={primary.label} onPress={primary.onPress} loading={primary.loading} disabled={primary.disabled} testID="onb-next" />
          {step === STEPS.length - 1 ? (
            <Button
              label="Plus tard"
              variant="ghost"
              loading={finishWithoutLocation.isPending}
              onPress={() => finishWithoutLocation.mutate()}
              testID="onb-skip-location"
            />
          ) : null}
        </View>
      }
    >
      <Header
        onBack={step > 0 && !(step <= 1 && profile) ? () => setStep((s) => s - 1) : undefined}
        right={step === 0 ? <Button label="Déconnexion" variant="ghost" onPress={() => void signOut()} /> : null}
      />
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.bar, { backgroundColor: i <= step ? colors.primary : colors.border }]} />
        ))}
      </View>
      <View style={styles.body}>{content}</View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: 6, marginBottom: spacing.lg },
  bar: { flex: 1, height: 4, borderRadius: radii.pill },
  body: { gap: spacing.md },
  subTitle: { marginTop: spacing.sm },
  footer: { gap: spacing.xs, paddingVertical: spacing.xs },
  locIcon: {
    width: 76,
    height: 76,
    borderRadius: 38,
    alignSelf: "center",
    alignItems: "center",
    justifyContent: "center",
    marginTop: spacing.xl
  }
});
