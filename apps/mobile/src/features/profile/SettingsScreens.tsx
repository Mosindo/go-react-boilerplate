import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi, profileApi, safetyApi } from "../../api/endpoints";
import type { Gender, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { keys, useBlocked, useMyProfile, usePreferences } from "../../hooks/queries";
import { validatePassword } from "../../lib/validation";
import type { RootParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { EmptyState, ErrorState, LoadingView } from "../../ui/Feedback";
import { ConfirmDialog } from "../../ui/Modal";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";
import { TextField } from "../../ui/TextField";
import { useToast } from "../../ui/Toast";
import { Stepper } from "./Stepper";
import { GenderChips, Section } from "./shared";

export function PreferencesScreen({ navigation }: NativeStackScreenProps<RootParamList, "Preferences">) {
  const query = usePreferences();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<Preferences | null>(null);
  const prefs = draft ?? query.data;

  const save = useMutation({
    mutationFn: (p: Preferences) => profileApi.updatePreferences(p),
    onSuccess: (p) => {
      queryClient.setQueryData(keys.preferences, p);
      void queryClient.invalidateQueries({ queryKey: keys.profile });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      toast.show("Préférences enregistrées.", "success");
      navigation.goBack();
    },
    onError: (e) => toast.show(e.message, "error")
  });

  if (query.isPending)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );
  if (query.isError || !prefs)
    return (
      <Screen>
        <ErrorState message={query.error?.message ?? "Erreur"} onRetry={() => void query.refetch()} />
      </Screen>
    );

  const update = (patch: Partial<Preferences>) => setDraft({ ...prefs, ...patch });
  const toggleGender = (g: Gender) =>
    update({
      interestedIn: prefs.interestedIn.includes(g)
        ? prefs.interestedIn.filter((x) => x !== g)
        : [...prefs.interestedIn, g]
    });

  return (
    <Screen scroll>
      <Section title="Je souhaite rencontrer">
        <GenderChips multi value={prefs.interestedIn} onToggle={toggleGender} />
      </Section>
      <Section title="Tranche d'âge">
        <Stepper
          label="Minimum"
          value={prefs.ageMin}
          min={18}
          max={prefs.ageMax}
          format={(v) => `${v} ans`}
          onChange={(v) => update({ ageMin: v })}
        />
        <Stepper
          label="Maximum"
          value={prefs.ageMax}
          min={prefs.ageMin}
          max={99}
          format={(v) => `${v} ans`}
          onChange={(v) => update({ ageMax: v })}
        />
      </Section>
      <Section title="Distance maximale">
        <Stepper
          label="Distance"
          value={prefs.maxDistanceKm}
          min={1}
          max={500}
          step={prefs.maxDistanceKm >= 50 ? 10 : 5}
          format={(v) => `${v} km`}
          onChange={(v) => update({ maxDistanceKm: v })}
        />
      </Section>
      <Text variant="caption" tone="textSubtle" style={styles.note}>
        Les critères s&apos;appliquent dans les deux sens : vous devez correspondre aux préférences l&apos;un
        de l&apos;autre.
      </Text>
      <Button
        title="Enregistrer"
        onPress={() => save.mutate(prefs)}
        loading={save.isPending}
        disabled={prefs.interestedIn.length === 0}
        testID="prefs-save"
      />
    </Screen>
  );
}

function SwitchRow({
  title,
  description,
  value,
  onChange,
  testID
}: {
  title: string;
  description: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={styles.switchRow}>
      <View style={styles.flex}>
        <Text variant="heading">{title}</Text>
        <Text tone="textMuted">{description}</Text>
      </View>
      <Switch
        value={value}
        onValueChange={onChange}
        accessibilityLabel={title}
        testID={testID}
        trackColor={{ true: colors.primary, false: colors.border }}
      />
    </View>
  );
}

export function PrivacyScreen() {
  const profile = useMyProfile();
  const blocked = useBlocked();
  const toast = useToast();
  const queryClient = useQueryClient();

  const update = useMutation({
    mutationFn: profileApi.update,
    onSuccess: (p) => {
      queryClient.setQueryData(keys.profile, p);
      void queryClient.invalidateQueries({ queryKey: keys.discover });
    },
    onError: (e) => toast.show(e.message, "error")
  });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.blocked });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
    },
    onError: (e) => toast.show(e.message, "error")
  });

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

  return (
    <Screen scroll>
      <Section title="Visibilité">
        <SwitchRow
          title="Apparaître dans la découverte"
          description="Désactive pour mettre ton profil en pause. Tes matchs te voient toujours."
          value={profile.data.discoverable}
          onChange={(v) => update.mutate({ discoverable: v })}
          testID="switch-discoverable"
        />
        <SwitchRow
          title="Afficher ma distance"
          description="Les autres voient une distance approximative, jamais ta position exacte."
          value={profile.data.showDistance}
          onChange={(v) => update.mutate({ showDistance: v })}
          testID="switch-distance"
        />
      </Section>
      <Section title="Personnes bloquées">
        {blocked.isPending ? (
          <LoadingView />
        ) : (blocked.data ?? []).length === 0 ? (
          <EmptyState icon="shield-checkmark-outline" title="Personne n'est bloqué" />
        ) : (
          (blocked.data ?? []).map((b) => (
            <View key={b.userId} style={styles.switchRow}>
              <Text variant="heading" style={styles.flex}>
                {b.firstName}
              </Text>
              <Button
                title="Débloquer"
                variant="secondary"
                onPress={() => unblock.mutate(b.userId)}
                loading={unblock.isPending && unblock.variables === b.userId}
              />
            </View>
          ))
        )}
      </Section>
    </Screen>
  );
}

export function SecurityScreen() {
  const { signOut } = useAuth();
  const toast = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [deletePassword, setDeletePassword] = useState("");

  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      setCurrent("");
      setNext("");
      toast.show("Mot de passe modifié.", "success");
    },
    onError: (e) =>
      setFieldError(e.message === "invalid credentials" ? "Mot de passe actuel incorrect." : e.message)
  });
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(deletePassword),
    onSuccess: () => {
      setDeleting(false);
      void signOut();
    },
    onError: (e) =>
      toast.show(e.message === "invalid credentials" ? "Mot de passe incorrect." : e.message, "error")
  });

  const submitChange = () => {
    const err = validatePassword(next);
    setFieldError(err);
    if (!err) change.mutate();
  };

  return (
    <Screen scroll>
      <Section title="Changer de mot de passe">
        <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureTextEntry />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          error={fieldError}
        />
        <Button
          title="Modifier"
          variant="secondary"
          onPress={submitChange}
          loading={change.isPending}
          disabled={!current || !next}
        />
      </Section>
      <Section title="Supprimer mon compte">
        <Text tone="textMuted" style={styles.note}>
          Ton profil, tes photos, tes matchs et tes messages seront supprimés définitivement. Cette action est
          irréversible.
        </Text>
        <Button
          title="Supprimer mon compte"
          variant="danger"
          onPress={() => setDeleting(true)}
          testID="delete-account"
        />
      </Section>
      <ConfirmDialog
        visible={deleting}
        title="Supprimer définitivement ?"
        message="Entre ton mot de passe pour confirmer."
        confirmLabel="Supprimer mon compte"
        destructive
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onCancel={() => setDeleting(false)}
      >
        <TextField
          label="Mot de passe"
          value={deletePassword}
          onChangeText={setDeletePassword}
          secureTextEntry
          testID="delete-password"
        />
      </ConfirmDialog>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  note: { marginBottom: spacing.md },
  switchRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm, marginBottom: spacing.md }
});
