import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi, profileApi, safetyApi } from "../../api/endpoints";
import type { Gender, Preferences } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { keys } from "../../state/queryClient";
import { useBlocks, useMyProfile, useSetMyProfile } from "../../state/queries";
import { spacing } from "../../theme/palette";
import {
  Button,
  Chip,
  Divider,
  EmptyState,
  ErrorState,
  Header,
  InlineError,
  ListRow,
  LoadingState,
  Screen,
  Section,
  Stepper,
  SwitchRow,
  Text,
  TextField,
  useToast
} from "../../ui";
import { genders, interestedInLabels } from "../../utils/labels";
import { deckEvents } from "../discovery/deckEvents";
import { readApproximateLocation } from "./media";
import { PhotoGrid } from "./PhotoGrid";

type P<K extends keyof MainStackParamList> = NativeStackScreenProps<MainStackParamList, K>;

export function PhotosScreen({ navigation }: P<"Photos">) {
  const { data } = useMyProfile();
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} subtitle="La première photo est votre photo principale." title="Mes photos" />
      {data ? <PhotoGrid photos={data.photos} /> : <LoadingState />}
    </Screen>
  );
}

export function PreferencesScreen(props: P<"Preferences">) {
  const { data } = useMyProfile();
  if (!data?.preferences) return <LoadingState />;
  return <PreferencesForm {...props} initial={data.preferences} />;
}

function PreferencesForm({ navigation, initial }: P<"Preferences"> & { initial: Preferences }) {
  const toast = useToast();
  const setMyProfile = useSetMyProfile();
  const [prefs, setPrefs] = useState(initial);
  const toggle = (g: Gender) =>
    setPrefs((p) => ({ ...p, interestedIn: p.interestedIn.includes(g) ? p.interestedIn.filter((x) => x !== g) : [...p.interestedIn, g] }));
  const save = useMutation({
    mutationFn: () => profileApi.updatePreferences(prefs),
    onSuccess: (saved) => {
      setMyProfile((prev) => ({ ...prev, preferences: saved }));
      deckEvents.markStale();
      toast({ title: "Préférences enregistrées", tone: "success" });
      navigation.goBack();
    }
  });
  return (
    <Screen
      footer={
        <View style={styles.footer}>
          <InlineError message={save.isError ? errorMessage(save.error) : null} />
          <Button disabled={!prefs.interestedIn.length} label="Enregistrer" loading={save.isPending} onPress={() => save.mutate()} size="lg" />
        </View>
      }
      scroll
    >
      <Header onBack={navigation.goBack} title="Préférences de rencontre" />
      <Section footer="Ces préférences s'appliquent dans les deux sens : vous ne voyez que les personnes qui souhaitent aussi vous rencontrer." title="Je souhaite rencontrer">
        <View style={styles.chips}>
          {genders.map((g) => (
            <Chip key={g} label={interestedInLabels[g]} onPress={() => toggle(g)} selected={prefs.interestedIn.includes(g)} />
          ))}
        </View>
      </Section>
      <Section title="Âge">
        <Stepper label="À partir de" max={prefs.maxAge} min={18} onChange={(minAge) => setPrefs((p) => ({ ...p, minAge }))} unit="ans" value={prefs.minAge} />
        <Divider />
        <Stepper label="Jusqu'à" max={99} min={prefs.minAge} onChange={(maxAge) => setPrefs((p) => ({ ...p, maxAge }))} unit="ans" value={prefs.maxAge} />
      </Section>
      <Section footer="Utilisée seulement si vous avez partagé votre position." title="Distance">
        <Stepper
          label="Jusqu'à"
          max={500}
          min={5}
          onChange={(maxDistanceKm) => setPrefs((p) => ({ ...p, maxDistanceKm }))}
          step={5}
          unit="km"
          value={prefs.maxDistanceKm}
        />
      </Section>
    </Screen>
  );
}

export function PrivacyScreen({ navigation }: P<"Privacy">) {
  const { data } = useMyProfile();
  const setMyProfile = useSetMyProfile();
  const toast = useToast();
  const privacy = useMutation({
    mutationFn: (patch: { discoverable?: boolean; showDistance?: boolean }) => profileApi.updatePrivacy(patch).then(() => patch),
    onSuccess: (patch) => {
      setMyProfile((prev) => (prev.profile ? { ...prev, profile: { ...prev.profile, ...patch } } : prev));
    },
    onError: (e) => toast({ title: "Modification impossible", body: errorMessage(e), tone: "error" })
  });
  const location = useMutation({
    mutationFn: async (mode: "update" | "clear") => {
      if (mode === "clear") {
        await profileApi.clearLocation();
        return { hasLocation: false, city: "" };
      }
      const loc = await readApproximateLocation();
      await profileApi.updateLocation(loc.latitude, loc.longitude, loc.city);
      return { hasLocation: true, city: loc.city };
    },
    onSuccess: (patch) => {
      setMyProfile((prev) => (prev.profile ? { ...prev, profile: { ...prev.profile, ...patch } } : prev));
      deckEvents.markStale();
      toast({ title: patch.hasLocation ? "Position mise à jour" : "Position supprimée", tone: "success" });
    },
    onError: (e) => toast({ title: "Localisation indisponible", body: errorMessage(e), tone: "error" })
  });
  const p = data?.profile;
  if (!p) return <LoadingState />;
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="Confidentialité et sécurité" />
      <Section footer="Masqué, votre profil n'apparaît plus dans Découvrir. Vos matchs existants restent actifs." title="Visibilité">
        <SwitchRow
          disabled={privacy.isPending}
          onChange={(discoverable) => privacy.mutate({ discoverable })}
          subtitle={p.discoverable ? "Votre profil est visible" : "Votre profil est masqué"}
          title="Apparaître dans Découvrir"
          value={p.discoverable}
        />
        <Divider />
        <SwitchRow disabled={privacy.isPending} onChange={(showDistance) => privacy.mutate({ showDistance })} title="Afficher ma distance" value={p.showDistance} />
      </Section>
      <Section footer="Seule une position arrondie (environ 1 km) est enregistrée. Les autres membres ne voient qu'une distance approximative." title="Localisation">
        <ListRow
          icon="navigate-outline"
          onPress={() => location.mutate("update")}
          subtitle={p.hasLocation ? p.city || "Position approximative enregistrée" : "Aucune position enregistrée"}
          title={location.isPending ? "Localisation…" : "Mettre à jour ma position"}
        />
        {p.hasLocation ? (
          <>
            <Divider />
            <ListRow destructive icon="close-circle-outline" onPress={() => location.mutate("clear")} title="Supprimer ma position" />
          </>
        ) : null}
      </Section>
      <Section title="Sécurité">
        <ListRow icon="hand-left-outline" onPress={() => navigation.navigate("BlockedUsers")} title="Membres bloqués" />
      </Section>
      <Text tone="subtle" variant="caption">
        Pour signaler ou bloquer quelqu’un, ouvrez son profil ou votre conversation puis touchez « … ».
      </Text>
    </Screen>
  );
}

export function BlockedUsersScreen({ navigation }: P<"BlockedUsers">) {
  const qc = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useBlocks();
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.blocks });
      deckEvents.markStale();
    }
  });
  let body: React.ReactNode;
  if (isLoading) body = <LoadingState />;
  else if (isError) body = <ErrorState message={errorMessage(error)} onRetry={refetch} />;
  else if (!data?.length) body = <EmptyState body="Les personnes que vous bloquez ne peuvent ni vous voir ni vous écrire." icon="hand-left-outline" title="Aucun membre bloqué" />;
  else
    body = (
      <Section>
        {data.map((b, i) => (
          <View key={b.userId}>
            {i > 0 ? <Divider /> : null}
            <View style={styles.blockRow}>
              <Text style={styles.flex}>{b.firstName || "Membre"}</Text>
              <Button label="Débloquer" loading={unblock.isPending && unblock.variables === b.userId} onPress={() => unblock.mutate(b.userId)} size="sm" variant="outline" />
            </View>
          </View>
        ))}
      </Section>
    );
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="Membres bloqués" />
      {body}
    </Screen>
  );
}

export function DeleteAccountScreen({ navigation }: P<"DeleteAccount">) {
  const { forgetSession } = useAuth();
  const toast = useToast();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      await forgetSession();
      toast({ title: "Compte supprimé", body: "Toutes vos données ont été effacées. Prenez soin de vous.", tone: "success" });
    }
  });
  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="Supprimer mon compte" />
      <View style={styles.form}>
        <Text>Cette action est définitive. Seront supprimés immédiatement :</Text>
        <Text tone="muted">• votre profil, vos photos et vos préférences{"\n"}• vos likes, vos matchs et toutes vos conversations{"\n"}• vos notifications et vos sessions</Text>
        <Text tone="muted" variant="caption">
          Si vous souhaitez simplement faire une pause, masquez plutôt votre profil dans Confidentialité.
        </Text>
        <TextField autoComplete="current-password" label="Confirmez avec votre mot de passe" onChangeText={setPassword} secureToggle testID="delete-password" value={password} />
        <InlineError message={remove.isError ? errorMessage(remove.error) : null} />
        <Button disabled={!password} label="Supprimer définitivement" loading={remove.isPending} onPress={() => remove.mutate()} testID="delete-confirm" variant="danger" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  footer: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs, padding: spacing.md },
  blockRow: { flexDirection: "row", alignItems: "center", padding: spacing.md, gap: spacing.sm },
  flex: { flex: 1 },
  form: { gap: spacing.md }
});
