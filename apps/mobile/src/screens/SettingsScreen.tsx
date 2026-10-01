import React, { useEffect, useState } from "react";
import { Modal, ScrollView, StyleSheet, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { changePassword, deleteAccount } from "../api/auth";
import { ApiError } from "../api/client";
import { profileApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { Preferences } from "../api/types";
import { PreferencesEditor } from "../components/PreferencesEditor";
import { useAuth } from "../hooks/useAuth";
import type { RootScreenProps } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Section } from "../shared/layout";
import { Button, Card, FormField, Input, ListItem, Notice, Text, radii, spacing, useTheme } from "../shared/ui";
import { passwordProblem } from "../utils/validation";

type Dialog = "password" | "delete" | null;

export default function SettingsScreen({ navigation }: RootScreenProps<"Settings">) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const queryClient = useQueryClient();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.getOwn });
  const prefsQuery = useQuery({ queryKey: queryKeys.preferences, queryFn: profileApi.getPreferences });
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const [dialog, setDialog] = useState<Dialog>(null);

  useEffect(() => {
    if (prefsQuery.data && !prefs) {
      setPrefs(prefsQuery.data);
    }
  }, [prefsQuery.data, prefs]);

  const savePrefs = useMutation({
    mutationFn: (value: Preferences) => profileApi.savePreferences(value),
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.preferences, saved);
      showToast("Préférences enregistrées.");
    }
  });

  const privacy = useMutation({
    mutationFn: (patch: { discoverable?: boolean; showDistance?: boolean }) => {
      const p = profile.data;
      if (!p) {
        throw new ApiError(400, "Profil indisponible.");
      }
      return profileApi.save({ firstName: p.firstName, gender: p.gender, bio: p.bio, city: p.city, ...patch });
    },
    onSuccess: (saved) => queryClient.setQueryData(queryKeys.profile, saved)
  });

  const clearLocation = useMutation({
    mutationFn: profileApi.clearLocation,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.profile });
      showToast("Position supprimée.");
    }
  });

  if (profile.isLoading || prefsQuery.isLoading || !prefs) {
    return profile.isError || prefsQuery.isError ? (
      <ErrorView
        message="Impossible de charger vos réglages."
        onAction={() => {
          void profile.refetch();
          void prefsQuery.refetch();
        }}
      />
    ) : (
      <LoadingView fullScreen />
    );
  }
  const p = profile.data;

  return (
    <ScrollView contentContainerStyle={styles.content}>
      <Section subtitle="Qui vous voyez dans la découverte." title="Préférences de rencontre">
        <PreferencesEditor onChange={setPrefs} value={prefs} />
        {savePrefs.isError ? (
          <Notice message={savePrefs.error instanceof ApiError ? savePrefs.error.message : "Enregistrement impossible."} tone="danger" />
        ) : null}
        <Button label="Enregistrer mes préférences" loading={savePrefs.isPending} onPress={() => savePrefs.mutate(prefs)} />
      </Section>

      <Section title="Confidentialité">
        <Card padded={false}>
          <ListItem
            right={
              <Switch
                accessibilityLabel="Apparaître dans la découverte"
                onValueChange={(value) => privacy.mutate({ discoverable: value })}
                trackColor={{ true: colors.primary, false: colors.borderStrong }}
                value={p?.discoverable ?? true}
              />
            }
            subtitle="Désactivé : seuls vos matchs vous voient"
            title="Apparaître dans la découverte"
          />
          <ListItem
            right={
              <Switch
                accessibilityLabel="Afficher ma distance"
                onValueChange={(value) => privacy.mutate({ showDistance: value })}
                trackColor={{ true: colors.primary, false: colors.borderStrong }}
                value={p?.showDistance ?? true}
              />
            }
            subtitle="Une distance arrondie, jamais votre position"
            title="Afficher ma distance"
          />
          <ListItem onPress={() => navigation.navigate("BlockedUsers")} subtitle="Voir et débloquer" title="Personnes bloquées" />
          {p?.hasLocation ? (
            <ListItem onPress={() => clearLocation.mutate()} subtitle="Efface la position enregistrée" title="Supprimer ma position" />
          ) : null}
        </Card>
      </Section>

      <Section title="Compte">
        <Card padded={false}>
          <ListItem onPress={() => setDialog("password")} title="Changer de mot de passe" />
          <ListItem onPress={() => void signOut()} title="Se déconnecter" />
          <ListItem
            destructive
            onPress={() => setDialog("delete")}
            subtitle="Supprime définitivement votre profil, vos photos et vos conversations"
            title="Supprimer mon compte"
          />
        </Card>
        <Text tone="subtle" variant="caption">
          Lumen est entièrement gratuit : pas d&apos;abonnement, pas de limite de likes, pas de publicité.
        </Text>
      </Section>

      <PasswordDialog onClose={() => setDialog(null)} visible={dialog === "password"} />
      <DeleteDialog onClose={() => setDialog(null)} visible={dialog === "delete"} />
    </ScrollView>
  );
}

function DialogShell({
  visible,
  title,
  children,
  onClose
}: {
  visible: boolean;
  title: string;
  children: React.ReactNode;
  onClose: () => void;
}) {
  const { colors } = useTheme();
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View accessibilityViewIsModal style={[styles.dialog, { backgroundColor: colors.surface }]}>
          <Text variant="heading">{title}</Text>
          {children}
        </View>
      </View>
    </Modal>
  );
}

function PasswordDialog({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const mutation = useMutation({
    mutationFn: () => changePassword(current, next),
    onSuccess: () => {
      showToast("Mot de passe modifié.");
      setCurrent("");
      setNext("");
      onClose();
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Modification impossible.")
  });
  const submit = () => {
    const problem = passwordProblem(next);
    if (!current) {
      setError("Saisissez votre mot de passe actuel.");
    } else if (problem) {
      setError(problem);
    } else {
      setError(null);
      mutation.mutate();
    }
  };
  return (
    <DialogShell onClose={onClose} title="Changer de mot de passe" visible={visible}>
      <FormField label="Mot de passe actuel">
        <Input accessibilityLabel="Mot de passe actuel" autoCapitalize="none" onChangeText={setCurrent} secureTextEntry value={current} />
      </FormField>
      <FormField hint="8 caractères minimum." label="Nouveau mot de passe">
        <Input accessibilityLabel="Nouveau mot de passe" autoCapitalize="none" onChangeText={setNext} secureTextEntry value={next} />
      </FormField>
      {error ? <Notice message={error} tone="danger" /> : null}
      <Button label="Modifier" loading={mutation.isPending} onPress={submit} />
      <Button label="Annuler" onPress={onClose} variant="ghost" />
    </DialogShell>
  );
}

function DeleteDialog({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { signOut } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const mutation = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      onClose();
      await signOut();
    },
    onError: (e) => setError(e instanceof ApiError ? e.message : "Suppression impossible.")
  });
  return (
    <DialogShell onClose={onClose} title="Supprimer mon compte" visible={visible}>
      <Text tone="muted">
        Cette action est définitive : profil, photos, matchs et messages seront effacés. Confirmez avec votre mot de passe.
      </Text>
      <Input
        accessibilityLabel="Mot de passe"
        autoCapitalize="none"
        onChangeText={setPassword}
        placeholder="Mot de passe"
        secureTextEntry
        value={password}
      />
      {error ? <Notice message={error} tone="danger" /> : null}
      <Button
        disabled={password.length === 0}
        label="Supprimer définitivement"
        loading={mutation.isPending}
        onPress={() => mutation.mutate()}
        variant="destructive"
      />
      <Button label="Annuler" onPress={onClose} variant="ghost" />
    </DialogShell>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg, paddingBottom: spacing.xxxl },
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  dialog: { width: "100%", maxWidth: 420, borderRadius: radii.xl, padding: spacing.xl, gap: spacing.md }
});
