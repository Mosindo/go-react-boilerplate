import React, { useMemo, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { Gender, LookingFor, MeProfile } from "../../api/types";
import { InterestPicker } from "../../components/InterestPicker";
import { OptionList } from "../../components/OptionList";
import { PhotoGrid } from "../../components/PhotoGrid";
import { keys, useInterests, useSavePreferences, useSaveProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { colors, radii, spacing } from "../../theme/tokens";
import { Button, Screen, Text, TextField, showToast } from "../../ui";
import { errorMessage } from "../../utils/errors";
import { ageFrom, formatBirthdate, genderLabels, interestedInLabels, lookingForLabels, maskBirthdate, parseBirthdate } from "../../utils/format";
import { LocationDeniedError, shareApproximateLocation } from "../../utils/location";

const STEPS = ["Toi", "Rencontres", "Photos", "Ta personnalité", "Localisation"] as const;

const genderOptions = (Object.keys(genderLabels) as Gender[]).map((g) => ({ value: g, label: genderLabels[g] }));
const interestedOptions = (Object.keys(interestedInLabels) as Gender[]).map((g) => ({ value: g, label: interestedInLabels[g] }));
const lookingOptions = (Object.keys(lookingForLabels) as LookingFor[]).map((l) => ({ value: l, label: lookingForLabels[l] }));

function initialStep(me: MeProfile): number {
  if (!me.completeness.hasProfile) {
    return 0;
  }
  if (!me.completeness.hasPreferences) {
    return 1;
  }
  if (!me.completeness.hasPhotos) {
    return 2;
  }
  return 3;
}

type Props = { me: MeProfile; onFinish: () => void };

export default function OnboardingScreen({ me, onFinish }: Props) {
  const qc = useQueryClient();
  const { signOut } = useAuth();
  const interests = useInterests();
  const saveProfile = useSaveProfile();
  const savePrefs = useSavePreferences();
  const [step, setStep] = useState(() => initialStep(me));
  const p = me.profile;

  const [firstName, setFirstName] = useState(p?.firstName ?? "");
  const [birthdate, setBirthdate] = useState(p ? formatBirthdate(p.birthdate) : "");
  const [gender, setGender] = useState<Gender | null>(p?.gender ?? null);
  const [interestedIn, setInterestedIn] = useState<Gender[]>(me.preferences?.interestedIn ?? []);
  const [lookingFor, setLookingFor] = useState<LookingFor>(p?.lookingFor ?? "unsure");
  const [bio, setBio] = useState(p?.bio ?? "");
  const [jobTitle, setJobTitle] = useState(p?.jobTitle ?? "");
  const [city, setCity] = useState(p?.city ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(p?.interests.map((i) => i.id) ?? []);
  const [locating, setLocating] = useState(false);

  const isoBirthdate = parseBirthdate(birthdate);
  const age = isoBirthdate ? ageFrom(isoBirthdate) : null;
  const birthError = birthdate.length === 10 && !isoBirthdate ? "Date invalide" : age !== null && age < 18 ? "Lueur est réservé aux 18 ans et plus" : null;

  const profileInput = useMemo(
    () => ({
      firstName: firstName.trim(),
      birthdate: isoBirthdate ?? "",
      gender: gender ?? "woman",
      bio,
      jobTitle,
      heightCm: p?.heightCm ?? null,
      lookingFor,
      city,
      interestIds
    }),
    [firstName, isoBirthdate, gender, bio, jobTitle, p?.heightCm, lookingFor, city, interestIds]
  );

  const photos = me.profile?.photos ?? [];
  const saving = saveProfile.isPending || savePrefs.isPending;

  const next = async () => {
    try {
      if (step === 1) {
        await saveProfile.mutateAsync(profileInput);
        const a = age ?? 30;
        await savePrefs.mutateAsync({
          interestedIn,
          ageMin: me.preferences?.ageMin ?? Math.max(18, a - 8),
          ageMax: me.preferences?.ageMax ?? Math.min(99, a + 8),
          maxDistanceKm: me.preferences?.maxDistanceKm ?? 50
        });
      }
      if (step === 3) {
        await saveProfile.mutateAsync(profileInput);
      }
      setStep((s) => s + 1);
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const finish = async (withLocation: boolean) => {
    if (withLocation) {
      setLocating(true);
      try {
        await shareApproximateLocation();
        showToast("Position approximative enregistrée", "success");
      } catch (err) {
        showToast(err instanceof LocationDeniedError ? err.message : "Impossible d'obtenir ta position. Tu pourras réessayer dans les réglages.", "error");
      } finally {
        setLocating(false);
      }
    }
    await qc.invalidateQueries({ queryKey: keys.me });
    onFinish();
  };

  const canContinue = [firstName.trim().length > 0 && !!isoBirthdate && !birthError, !!gender && interestedIn.length > 0, photos.length > 0, true, true][step];

  const toggleInterested = (g: Gender) => setInterestedIn((cur) => (cur.includes(g) ? cur.filter((x) => x !== g) : [...cur, g]));

  return (
    <Screen
      scroll
      keyboard
      edges={["top"]}
      testID={`onboarding-step-${step}`}
      footer={
        step < 4 ? (
          <View style={styles.footerRow}>
            {step > 0 ? <Button label="Retour" variant="ghost" onPress={() => setStep((s) => s - 1)} style={styles.back} /> : null}
            <Button label="Continuer" onPress={next} disabled={!canContinue} loading={saving} style={styles.grow} testID="onboarding-next" />
          </View>
        ) : (
          <View style={styles.finish}>
            <Button label="Partager ma position approximative" icon="navigate" onPress={() => finish(true)} loading={locating} testID="onboarding-location" />
            <Button label="Plus tard" variant="ghost" onPress={() => finish(false)} testID="onboarding-skip-location" />
          </View>
        )
      }
    >
      <View style={styles.progress} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.dot, i <= step && styles.dotOn]} />
        ))}
      </View>
      <Text variant="label" tone="accent">
        ÉTAPE {step + 1} · {STEPS[step].toUpperCase()}
      </Text>

      {step === 0 ? (
        <View style={styles.section}>
          <Text variant="title">Faisons connaissance</Text>
          <TextField label="Prénom" value={firstName} onChangeText={setFirstName} maxLength={40} autoComplete="given-name" testID="onboarding-firstname" />
          <TextField
            label="Date de naissance"
            value={birthdate}
            onChangeText={(v) => setBirthdate(maskBirthdate(v))}
            placeholder="JJ/MM/AAAA"
            keyboardType="number-pad"
            error={birthError}
            hint="Seul ton âge sera visible."
            testID="onboarding-birthdate"
          />
        </View>
      ) : null}

      {step === 1 ? (
        <View style={styles.section}>
          <Text variant="title">Qui es-tu, qui cherches-tu ?</Text>
          <Text variant="label" tone="muted">
            JE SUIS
          </Text>
          <OptionList options={genderOptions} selected={gender ? [gender] : []} onToggle={setGender} testIDPrefix="gender" />
          <Text variant="label" tone="muted">
            JE SOUHAITE RENCONTRER
          </Text>
          <OptionList options={interestedOptions} selected={interestedIn} onToggle={toggleInterested} multiple testIDPrefix="interested" />
          <Text variant="label" tone="muted">
            JE CHERCHE
          </Text>
          <OptionList options={lookingOptions} selected={[lookingFor]} onToggle={setLookingFor} testIDPrefix="looking" />
        </View>
      ) : null}

      {step === 2 ? (
        <View style={styles.section}>
          <Text variant="title">Montre-toi</Text>
          <Text tone="muted">
            Ajoute au moins une photo (jusqu&apos;à 6). La première sera ta photo principale. Les métadonnées (dont la position GPS) sont supprimées.
          </Text>
          <PhotoGrid photos={photos} />
        </View>
      ) : null}

      {step === 3 ? (
        <View style={styles.section}>
          <Text variant="title">Ce qui te ressemble</Text>
          <TextField
            label="Bio"
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
            counter
            placeholder="Ce qui te fait vibrer, ce que tu cherches…"
            testID="onboarding-bio"
          />
          <TextField label="Métier (facultatif)" value={jobTitle} onChangeText={setJobTitle} maxLength={60} />
          <TextField label="Ville ou quartier (facultatif)" value={city} onChangeText={setCity} maxLength={80} hint="Affiché tel quel sur ton profil." />
          <Text variant="label" tone="muted">
            CENTRES D&apos;INTÉRÊT
          </Text>
          <InterestPicker interests={interests.data} selected={interestIds} onChange={setInterestIds} />
        </View>
      ) : null}

      {step === 4 ? (
        <View style={styles.section}>
          <Text variant="title">Des rencontres près de toi</Text>
          <View style={styles.privacy}>
            <Text variant="bodyStrong">Ta vie privée d&apos;abord</Text>
            <Text tone="muted">• Ta position est arrondie à environ 1 km avant d&apos;être enregistrée.</Text>
            <Text tone="muted">• Personne ne voit ta position : seulement une distance approximative.</Text>
            <Text tone="muted">• Elle n&apos;est mise à jour que lorsque tu le demandes.</Text>
          </View>
          <Text tone="muted">Sans position, tu verras des profils sans filtre de distance.</Text>
        </View>
      ) : null}

      <Button label="Se déconnecter" variant="ghost" compact onPress={signOut} style={styles.signOut} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.xs, marginBottom: spacing.lg },
  dot: { flex: 1, height: 4, borderRadius: 2, backgroundColor: colors.border },
  dotOn: { backgroundColor: colors.accent },
  section: { gap: spacing.lg, marginTop: spacing.md },
  footerRow: { flexDirection: "row", gap: spacing.sm },
  back: { minWidth: 100 },
  grow: { flex: 1 },
  finish: { gap: spacing.sm },
  privacy: { gap: spacing.sm, padding: spacing.lg, borderRadius: radii.lg, backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border },
  signOut: { marginTop: spacing.xxxl, alignSelf: "center" }
});
