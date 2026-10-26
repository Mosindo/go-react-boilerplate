import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authApi, profileApi, safetyApi } from "../api";
import { useAuth } from "../auth/AuthContext";
import { Button } from "../components/Button";
import { useFeedback } from "../components/Feedback";
import { LocationPicker } from "../components/LocationPicker";
import { PhotoManager } from "../components/PhotoManager";
import { Avatar } from "../components/Photo";
import { PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm } from "../components/ProfileForm";
import { Screen } from "../components/Screen";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { Text } from "../components/Text";
import { TextField } from "../components/TextField";
import { useOwnProfile } from "../hooks/useQueries";
import { validPassword } from "../lib/format";
import { queryKeys } from "../lib/queryKeys";
import { spacing, useTheme } from "../theme";

const frame = { maxWidth: 560, width: "100%", alignSelf: "center", gap: spacing.xl } as const;

function useProfileGate() {
  const profile = useOwnProfile();
  if (profile.isPending)
    return {
      node: (
        <Screen>
          <LoadingState />
        </Screen>
      ),
      data: null
    };
  if (profile.isError || !profile.data)
    return {
      node: (
        <Screen>
          <ErrorState onRetry={() => void profile.refetch()} />
        </Screen>
      ),
      data: null
    };
  return { node: null, data: profile.data };
}

export function EditProfileScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const { toast } = useFeedback();
  const gate = useProfileGate();
  const [error, setError] = useState<string | null>(null);
  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      toast("Profil mis à jour", "success");
      navigation.goBack();
    },
    onError: (err) => setError(err instanceof Error ? err.message : "Enregistrement impossible.")
  });
  if (!gate.data) return gate.node;
  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      <ProfileForm
        initial={gate.data}
        submitLabel="Enregistrer"
        loading={save.isPending}
        serverError={error}
        onSubmit={(i) => save.mutate(i)}
      />
    </Screen>
  );
}

export function PhotosScreen() {
  const gate = useProfileGate();
  if (!gate.data) return gate.node;
  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      <Text muted>
        La première photo est votre photo principale. Les photos sont recompressées et nettoyées de leurs
        métadonnées (position GPS, appareil).
      </Text>
      <PhotoManager photos={gate.data.photos} />
    </Screen>
  );
}

export function PreferencesScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const { toast } = useFeedback();
  const gate = useProfileGate();
  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (saved) => {
      queryClient.setQueryData(queryKeys.profile, saved);
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      toast("Préférences enregistrées", "success");
      navigation.goBack();
    },
    onError: (err) => toast(err instanceof Error ? err.message : "Enregistrement impossible.", "error")
  });
  if (!gate.data) return gate.node;
  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      <PreferencesForm
        initial={gate.data.preferences}
        submitLabel="Enregistrer"
        loading={save.isPending}
        onSubmit={(p) => save.mutate(p)}
      />
    </Screen>
  );
}

function ToggleRow({
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
  testID: string;
}) {
  const theme = useTheme();
  return (
    <View style={styles.toggle}>
      <View style={styles.toggleText}>
        <Text variant="bodyStrong">{title}</Text>
        <Text variant="caption" muted>
          {description}
        </Text>
      </View>
      <Switch
        value={value}
        onValueChange={onChange}
        accessibilityLabel={title}
        testID={testID}
        trackColor={{ true: theme.primary, false: theme.border }}
      />
    </View>
  );
}

export function PrivacyScreen() {
  const queryClient = useQueryClient();
  const { toast } = useFeedback();
  const gate = useProfileGate();
  const save = useMutation({
    mutationFn: ({ visible, distance }: { visible: boolean; distance: boolean }) =>
      profileApi.savePrivacy(visible, distance),
    onSuccess: (saved) => queryClient.setQueryData(queryKeys.profile, saved),
    onError: (err) => toast(err instanceof Error ? err.message : "Enregistrement impossible.", "error")
  });
  if (!gate.data) return gate.node;
  const p = gate.data;
  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      <ToggleRow
        title="Profil visible"
        description="Désactivez pour mettre votre profil en pause : vous n'apparaissez plus dans la découverte, vos matchs restent actifs."
        value={p.isVisible}
        onChange={(v) => save.mutate({ visible: v, distance: p.showDistance })}
        testID="toggle-visible"
      />
      <ToggleRow
        title="Afficher ma distance"
        description="Les autres voient une distance approximative (par paliers de 5 km). Désactivée, aucune distance n'est affichée."
        value={p.showDistance}
        onChange={(v) => save.mutate({ visible: p.isVisible, distance: v })}
        testID="toggle-distance"
      />
      <View style={styles.section}>
        <Text variant="heading">Ma position</Text>
        <LocationPicker initialCity={p.city} hasLocation={p.hasLocation} />
      </View>
    </Screen>
  );
}

export function BlockedScreen() {
  const queryClient = useQueryClient();
  const { toast } = useFeedback();
  const blocks = useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      toast("Personne débloquée", "success");
    },
    onError: (err) => toast(err instanceof Error ? err.message : "Action impossible.", "error")
  });
  if (blocks.isPending)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (blocks.isError)
    return (
      <Screen>
        <ErrorState message={blocks.error.message} onRetry={() => void blocks.refetch()} />
      </Screen>
    );
  if (blocks.data.length === 0)
    return (
      <Screen>
        <EmptyState
          emoji="🛡️"
          title="Aucune personne bloquée"
          message="Les personnes que vous bloquez apparaissent ici."
        />
      </Screen>
    );
  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      {blocks.data.map((b) => (
        <View key={b.userId} style={styles.blockRow}>
          <Avatar url={b.photoUrl} name={b.firstName || "?"} size={48} />
          <Text variant="bodyStrong" style={styles.flex}>
            {b.firstName || "Compte supprimé"}
          </Text>
          <Button label="Débloquer" variant="secondary" small onPress={() => unblock.mutate(b.userId)} />
        </View>
      ))}
    </Screen>
  );
}

export function AccountScreen() {
  const theme = useTheme();
  const { user, signOut, afterAccountDeleted } = useAuth();
  const { toast, confirm } = useFeedback();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [pwError, setPwError] = useState<string | null>(null);
  const [deletePassword, setDeletePassword] = useState("");
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      setCurrent("");
      setNext("");
      setPwError(null);
      toast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
    },
    onError: (err) => setPwError(err instanceof Error ? err.message : "Modification impossible.")
  });

  const submitChange = () => {
    const issue = validPassword(next);
    if (!current) return setPwError("Entrez votre mot de passe actuel.");
    if (issue) return setPwError(issue);
    setPwError(null);
    change.mutate();
  };

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(deletePassword),
    onSuccess: () => afterAccountDeleted(),
    onError: (err) => setDeleteError(err instanceof Error ? err.message : "Suppression impossible.")
  });

  const submitDelete = async () => {
    if (!deletePassword) return setDeleteError("Entrez votre mot de passe pour confirmer.");
    setDeleteError(null);
    const ok = await confirm({
      title: "Supprimer définitivement mon compte ?",
      message:
        "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.",
      confirmLabel: "Supprimer mon compte",
      destructive: true
    });
    if (ok) remove.mutate();
  };

  return (
    <Screen scroll edges={["left", "right"]} contentStyle={frame}>
      <View style={styles.section}>
        <Text variant="heading">Connecté avec</Text>
        <Text muted>{user?.email}</Text>
        <Button label="Se déconnecter" variant="secondary" onPress={() => void signOut()} testID="logout" />
      </View>

      <View style={styles.section}>
        <Text variant="heading">Changer de mot de passe</Text>
        <TextField
          label="Mot de passe actuel"
          value={current}
          onChangeText={setCurrent}
          secure
          autoCapitalize="none"
          testID="current-password"
        />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secure
          autoCapitalize="none"
          error={pwError}
          testID="new-password"
        />
        <Button
          label="Modifier le mot de passe"
          onPress={submitChange}
          loading={change.isPending}
          testID="change-password"
        />
      </View>

      <View style={styles.section}>
        <Text variant="heading" color={theme.danger}>
          Zone sensible
        </Text>
        <Text muted>La suppression de votre compte efface toutes vos données de nos serveurs.</Text>
        <TextField
          label="Mot de passe"
          value={deletePassword}
          onChangeText={setDeletePassword}
          secure
          autoCapitalize="none"
          error={deleteError}
          testID="delete-password"
        />
        <Button
          label="Supprimer mon compte"
          variant="danger"
          onPress={() => void submitDelete()}
          loading={remove.isPending}
          testID="delete-account"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  toggle: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  toggleText: { flex: 1, gap: 2 },
  section: { gap: spacing.md },
  blockRow: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  flex: { flex: 1 }
});
