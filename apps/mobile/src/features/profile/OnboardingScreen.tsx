import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { Gender, MyProfile, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { keys, useInterests, useMyProfile } from "../../hooks/queries";
import { geocodeCity, locateMe } from "../../lib/device";
import { useOnboarding } from "../../navigation/RootNavigator";
import { radii, spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { Chip } from "../../ui/Chip";
import { ErrorState, LoadingView } from "../../ui/Feedback";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";
import { useToast } from "../../ui/Toast";
import { PhotoManager } from "./PhotoManager";
import { Stepper } from "./Stepper";
import { GenderChips } from "./shared";

const STEPS = ["gender", "preferences", "photos", "location", "about"] as const;
type Step = (typeof STEPS)[number];

function firstStep(profile: MyProfile): number {
  if (profile.missing.includes("gender")) return 0;
  if (profile.missing.includes("interestedIn")) return 1;
  if (profile.missing.includes("photos")) return 2;
  if (profile.missing.includes("location")) return 3;
  return 4;
}

const TITLES: Record<Step, { title: string; subtitle: string }> = {
  gender: { title: "Comment te présentes-tu ?", subtitle: "Cette information apparaît sur ton profil." },
  preferences: {
    title: "Qui as-tu envie de rencontrer ?",
    subtitle: "Tu pourras changer ces critères à tout moment."
  },
  photos: {
    title: "Ajoute tes plus belles photos",
    subtitle: "Au moins une photo est nécessaire. La première sera ta photo principale."
  },
  location: {
    title: "Où te trouves-tu ?",
    subtitle: "Pour te proposer des personnes proches. Seule une distance approximative est affichée."
  },
  about: {
    title: "Parle-nous un peu de toi",
    subtitle: "Une courte bio et quelques centres d'intérêt donnent envie de te parler."
  }
};

function Flow({ profile }: { profile: MyProfile }) {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const onboarding = useOnboarding();
  const interests = useInterests();
  const [index, setIndex] = useState(() => firstStep(profile));
  const step = STEPS[index];

  const [gender, setGender] = useState<Gender | undefined>(profile.gender);
  const [prefs, setPrefs] = useState<Preferences>({
    interestedIn: [],
    ageMin: 18,
    ageMax: 45,
    maxDistanceKm: 50
  });
  const [city, setCity] = useState(profile.city);
  const [bio, setBio] = useState(profile.bio);
  const [picked, setPicked] = useState<string[]>(profile.interests.map((i) => i.slug));

  const refreshProfile = (p: MyProfile) => queryClient.setQueryData(keys.profile, p);
  const next = () => setIndex((i) => Math.min(STEPS.length - 1, i + 1));
  const onError = (e: Error) => toast.show(e.message, "error");

  const saveGender = useMutation({
    mutationFn: () => profileApi.update({ gender }),
    onSuccess: (p) => {
      refreshProfile(p);
      next();
    },
    onError
  });
  const savePrefs = useMutation({
    mutationFn: () => profileApi.updatePreferences(prefs),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.preferences });
      next();
    },
    onError
  });
  const locate = useMutation({
    mutationFn: async (manual: boolean) => {
      const where = manual ? await geocodeCity(city) : await locateMe();
      return profileApi.setLocation(where.latitude, where.longitude, where.city ?? (city || undefined));
    },
    onSuccess: (p) => {
      refreshProfile(p);
      setCity(p.city);
      next();
    },
    onError
  });
  const finish = useMutation({
    mutationFn: () => profileApi.update({ bio: bio.trim(), interests: picked }),
    onSuccess: (p) => {
      refreshProfile(p);
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      onboarding.finish();
    },
    onError
  });

  const toggleInterest = (slug: string) =>
    setPicked((cur) =>
      cur.includes(slug) ? cur.filter((s) => s !== slug) : cur.length >= 8 ? cur : [...cur, slug]
    );
  const togglePref = (g: Gender) =>
    setPrefs((p) => ({
      ...p,
      interestedIn: p.interestedIn.includes(g)
        ? p.interestedIn.filter((x) => x !== g)
        : [...p.interestedIn, g]
    }));

  const { title, subtitle } = TITLES[step];
  return (
    <Screen scroll>
      <View style={styles.progress} accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View
            key={s}
            style={[styles.bar, { backgroundColor: i <= index ? colors.primary : colors.border }]}
          />
        ))}
      </View>
      <Text variant="title">{title}</Text>
      <Text tone="textMuted" style={styles.subtitle}>
        {subtitle}
      </Text>

      {step === "gender" ? (
        <>
          <GenderChips value={gender ? [gender] : []} onToggle={setGender} />
          <Button
            title="Continuer"
            onPress={() => saveGender.mutate()}
            disabled={!gender}
            loading={saveGender.isPending}
            style={styles.cta}
            testID="onboarding-next"
          />
        </>
      ) : null}

      {step === "preferences" ? (
        <>
          <GenderChips multi value={prefs.interestedIn} onToggle={togglePref} />
          <View style={styles.steppers}>
            <Stepper
              label="Âge minimum"
              value={prefs.ageMin}
              min={18}
              max={prefs.ageMax}
              format={(v) => `${v} ans`}
              onChange={(v) => setPrefs((p) => ({ ...p, ageMin: v }))}
            />
            <Stepper
              label="Âge maximum"
              value={prefs.ageMax}
              min={prefs.ageMin}
              max={99}
              format={(v) => `${v} ans`}
              onChange={(v) => setPrefs((p) => ({ ...p, ageMax: v }))}
            />
            <Stepper
              label="Distance max"
              value={prefs.maxDistanceKm}
              min={5}
              max={500}
              step={prefs.maxDistanceKm >= 50 ? 10 : 5}
              format={(v) => `${v} km`}
              onChange={(v) => setPrefs((p) => ({ ...p, maxDistanceKm: v }))}
            />
          </View>
          <Button
            title="Continuer"
            onPress={() => savePrefs.mutate()}
            disabled={prefs.interestedIn.length === 0}
            loading={savePrefs.isPending}
            style={styles.cta}
            testID="onboarding-next"
          />
        </>
      ) : null}

      {step === "photos" ? (
        <>
          <PhotoManager photos={profile.photos} />
          <Button
            title="Continuer"
            onPress={next}
            disabled={profile.photos.length === 0}
            style={styles.cta}
            testID="onboarding-next"
          />
        </>
      ) : null}

      {step === "location" ? (
        <>
          <Button
            title="Utiliser ma position"
            icon="locate"
            onPress={() => locate.mutate(false)}
            loading={locate.isPending && locate.variables === false}
            testID="onboarding-locate"
          />
          <Text tone="textMuted" align="center" style={styles.or}>
            ou
          </Text>
          <TextField label="Ma ville" value={city} onChangeText={setCity} testID="onboarding-city" />
          <Button
            title="Utiliser cette ville"
            variant="secondary"
            onPress={() => locate.mutate(true)}
            disabled={!city.trim()}
            loading={locate.isPending && locate.variables === true}
          />
        </>
      ) : null}

      {step === "about" ? (
        <>
          <TextField
            label="Bio (facultatif)"
            value={bio}
            onChangeText={setBio}
            multiline
            maxLength={500}
            helper={`${bio.length}/500`}
            testID="onboarding-bio"
          />
          <Text variant="label" tone="textMuted" style={styles.interestsTitle}>
            Centres d&apos;intérêt ({picked.length}/8)
          </Text>
          <View style={styles.chips}>
            {(interests.data ?? []).map((i) => (
              <Chip
                key={i.slug}
                label={i.label}
                selected={picked.includes(i.slug)}
                onPress={() => toggleInterest(i.slug)}
              />
            ))}
          </View>
          <Button
            title="Terminer"
            onPress={() => finish.mutate()}
            loading={finish.isPending}
            style={styles.cta}
            testID="onboarding-finish"
          />
        </>
      ) : null}

      <View style={styles.footer}>
        {index > 0 && step !== "about" ? (
          <Button title="Retour" variant="ghost" onPress={() => setIndex(index - 1)} />
        ) : null}
        <Button title="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}

export function OnboardingScreen() {
  const profile = useMyProfile();
  if (profile.isPending)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (profile.isError)
    return (
      <Screen>
        <ErrorState message={profile.error.message} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  return <Flow profile={profile.data} />;
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: 6, marginBottom: spacing.lg, marginTop: spacing.xs },
  bar: { flex: 1, height: 4, borderRadius: radii.pill },
  subtitle: { marginTop: spacing.xs, marginBottom: spacing.lg },
  steppers: { marginTop: spacing.md },
  cta: { marginTop: spacing.lg },
  or: { marginVertical: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  interestsTitle: { marginBottom: spacing.xs },
  footer: { marginTop: spacing.md }
});
