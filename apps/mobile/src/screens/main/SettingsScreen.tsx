import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { Alert, ScrollView, StyleSheet, Text, View } from "react-native";
import { deleteAccount } from "../../api/auth";
import { saveProfile } from "../../api/profile";
import type { OwnProfile, ProfileInput } from "../../api/types";
import { Button } from "../../components/Button";
import { ListRow } from "../../components/ListRow";
import { Sheet } from "../../components/Sheet";
import { ErrorView, LoadingView } from "../../components/StateViews";
import { TextField } from "../../components/TextField";
import { queryKeys, useProfile } from "../../hooks/queries";
import { errorMessage, useAsyncAction } from "../../hooks/useAsyncAction";
import type { RootStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { bumpDiscoverVersion } from "../../state/discoverStore";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "Settings">;

function toInput(profile: OwnProfile, overrides: Partial<ProfileInput>): ProfileInput {
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    interestIds: profile.interests.map((i) => i.id),
    isVisible: profile.isVisible,
    showDistance: profile.showDistance,
    ...overrides
  };
}

export default function SettingsScreen({ navigation }: Props) {
  const theme = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { signOut, afterAccountDeleted } = useAuth();
  const query = useProfile();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [password, setPassword] = useState("");

  const update = async (overrides: Partial<ProfileInput>) => {
    if (!query.data) return;
    try {
      const saved = await saveProfile(toInput(query.data, overrides));
      queryClient.setQueryData(queryKeys.profile, saved);
      bumpDiscoverVersion();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const remove = useAsyncAction(async () => {
    if (!password) {
      throw new Error("Entrez votre mot de passe pour confirmer.");
    }
    await deleteAccount(password);
    await afterAccountDeleted();
  });

  if (query.isLoading) return <LoadingView />;
  if (query.isError || !query.data) return <ErrorView onRetry={() => void query.refetch()} />;
  const profile = query.data;

  const confirmSignOut = () =>
    Alert.alert("Se déconnecter ?", undefined, [
      { text: "Annuler", style: "cancel" },
      { text: "Se déconnecter", onPress: () => void signOut() }
    ]);

  return (
    <ScrollView
      contentContainerStyle={styles.content}
      style={{ backgroundColor: theme.background }}
    >
      <Text style={[styles.section, { color: theme.textMuted }]}>Confidentialité</Text>
      <View style={[styles.group, { backgroundColor: theme.surface, borderColor: theme.border }]}>
        <ListRow
          icon="eye-off-outline"
          onSwitchChange={(visible) => void update({ isVisible: visible })}
          subtitle="Désactivez pour mettre votre profil en pause : seuls vos matchs vous voient."
          switchValue={profile.isVisible}
          title="Visible dans la découverte"
        />
        <ListRow
          icon="location-outline"
          onSwitchChange={(show) => void update({ showDistance: show })}
          subtitle="Une distance approximative, jamais votre position exacte."
          switchValue={profile.showDistance}
          title="Afficher ma distance"
        />
        <ListRow
          icon="ban-outline"
          onPress={() => navigation.navigate("Blocked")}
          title="Utilisateurs bloqués"
        />
      </View>

      <Text style={[styles.section, { color: theme.textMuted }]}>Compte</Text>
      <View style={[styles.group, { backgroundColor: theme.surface, borderColor: theme.border }]}>
        <ListRow
          icon="key-outline"
          onPress={() => navigation.navigate("ChangePassword")}
          title="Changer le mot de passe"
        />
        <ListRow icon="log-out-outline" onPress={confirmSignOut} title="Se déconnecter" />
        <ListRow
          destructive
          icon="trash-outline"
          onPress={() => setDeleteOpen(true)}
          title="Supprimer mon compte"
        />
      </View>

      <Sheet
        onClose={() => {
          setDeleteOpen(false);
          setPassword("");
        }}
        title="Supprimer mon compte"
        visible={deleteOpen}
      >
        <Text style={[styles.warning, { color: theme.textMuted }]}>
          Cette action est définitive : votre profil, vos photos, vos matchs et vos messages seront
          supprimés immédiatement.
        </Text>
        <TextField
          label="Mot de passe"
          onChangeText={setPassword}
          revealable
          secureTextEntry
          value={password}
        />
        {remove.error ? (
          <Text style={[styles.error, { color: theme.danger }]}>{remove.error}</Text>
        ) : null}
        <Button
          loading={remove.loading}
          onPress={() => void remove.run()}
          testID="delete-account-confirm"
          title="Supprimer définitivement"
          variant="danger"
        />
      </Sheet>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg },
  section: {
    fontSize: fontSizes.xs,
    fontWeight: "700",
    letterSpacing: 0.8,
    textTransform: "uppercase",
    marginBottom: spacing.sm,
    marginTop: spacing.lg
  },
  group: { borderWidth: 1, borderRadius: 20, overflow: "hidden" },
  warning: { fontSize: fontSizes.sm, lineHeight: 20, marginBottom: spacing.lg },
  error: { fontSize: fontSizes.sm, marginBottom: spacing.md }
});
