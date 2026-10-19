import React, { useState } from "react";
import { Alert, FlatList, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { deleteAccount } from "../../api/auth";
import { saveProfile } from "../../api/profile";
import { listBlocks, reportUser, unblockUser } from "../../api/social";
import type { ReportReason } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/errors";
import { profileToInput } from "../../lib/profile";
import type { MainStackParams } from "../../navigation/types";
import {
  Avatar,
  Button,
  Chip,
  EmptyState,
  ErrorState,
  Input,
  Loading,
  Screen,
  Text,
  useToast
} from "../../ui";
import { spacing } from "../../theme/tokens";

export const REPORT_REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export function ReportScreen({ navigation, route }: NativeStackScreenProps<MainStackParams, "Report">) {
  const { userId, name, defaultReason } = route.params;
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(defaultReason ?? null);
  const [details, setDetails] = useState("");
  const mutation = useMutation({
    mutationFn: () => reportUser(userId, reason as ReportReason, details),
    onSuccess: () => {
      toast.show("Merci, votre signalement a été transmis.", "success");
      navigation.goBack();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  return (
    <Screen scroll testID="report-screen">
      <Text variant="title">Signaler {name}</Text>
      <Text tone="muted">Votre signalement est confidentiel. {name} ne sera pas informé(e).</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {REPORT_REASONS.map((r) => (
          <Chip
            key={r.value}
            label={r.label}
            selected={reason === r.value}
            onPress={() => setReason(r.value)}
            testID={`reason-${r.value}`}
          />
        ))}
      </View>
      <Input
        label="Détails (facultatif)"
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
        hint={`${details.length}/1000`}
      />
      <Button
        label="Envoyer le signalement"
        onPress={() => mutation.mutate()}
        disabled={!reason}
        loading={mutation.isPending}
        testID="report-submit"
      />
    </Screen>
  );
}

export function BlockedUsersScreen() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["blocks"], queryFn: listBlocks });
  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  if (query.data.length === 0) {
    return (
      <EmptyState
        icon="🛡"
        title="Aucun utilisateur bloqué"
        message="Les personnes que vous bloquez apparaîtront ici."
      />
    );
  }
  return (
    <FlatList
      data={query.data}
      keyExtractor={(b) => b.userId}
      contentContainerStyle={{ padding: spacing.lg, gap: spacing.md }}
      renderItem={({ item }) => (
        <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
          <Avatar photoId={item.photoId} name={item.firstName} size={48} />
          <Text style={{ flex: 1 }} bold>
            {item.firstName || "Utilisateur"}
          </Text>
          <Button
            label="Débloquer"
            variant="secondary"
            onPress={() => {
              unblockUser(item.userId)
                .then(() => queryClient.invalidateQueries({ queryKey: ["blocks"] }))
                .catch((e) => toast.show(errorMessage(e), "error"));
            }}
          />
        </View>
      )}
    />
  );
}

export function SettingsScreen({ navigation }: NativeStackScreenProps<MainStackParams, "Settings">) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const { user, signOut, onAccountDeleted } = useAuth();
  const profile = useOwnProfile();
  const [confirming, setConfirming] = useState(false);
  const [password, setPassword] = useState("");
  const [deleting, setDeleting] = useState(false);

  const toggleVisible = async (visible: boolean) => {
    if (!profile.data) return;
    try {
      const saved = await saveProfile(profileToInput(profile.data, { isVisible: visible }));
      queryClient.setQueryData(["profile"], saved);
    } catch (e) {
      toast.show(errorMessage(e), "error");
    }
  };

  const remove = async () => {
    setDeleting(true);
    try {
      await deleteAccount(password);
      onAccountDeleted();
    } catch (e) {
      toast.show(errorMessage(e), "error");
      setDeleting(false);
    }
  };

  return (
    <Screen scroll testID="settings-screen">
      <Text variant="caption" tone="muted">
        Connecté en tant que {user?.email}
      </Text>

      <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md }}>
        <View style={{ flex: 1 }}>
          <Text bold>Afficher mon profil</Text>
          <Text variant="caption" tone="muted">
            Désactivez pour faire une pause : vous n&apos;apparaissez plus dans la découverte, vos matchs
            restent actifs.
          </Text>
        </View>
        <Switch
          accessibilityLabel="Afficher mon profil"
          value={profile.data?.isVisible ?? true}
          disabled={!profile.data}
          onValueChange={(v) => void toggleVisible(v)}
          testID="visibility-switch"
        />
      </View>

      <Button
        label="Utilisateurs bloqués"
        variant="secondary"
        onPress={() => navigation.navigate("BlockedUsers")}
      />
      <Button
        label="Se déconnecter"
        variant="secondary"
        onPress={() => void signOut()}
        testID="logout-button"
      />

      <View style={{ gap: spacing.md, marginTop: spacing.xl }}>
        <Text variant="subtitle" tone="danger">
          Zone sensible
        </Text>
        {confirming ? (
          <>
            <Text tone="muted">
              La suppression est définitive : profil, photos, likes, matchs et messages seront effacés pour
              vous comme pour vos matchs. Confirmez avec votre mot de passe.
            </Text>
            <Input
              label="Mot de passe"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              testID="delete-password"
            />
            <Button
              label="Supprimer définitivement mon compte"
              variant="danger"
              loading={deleting}
              disabled={!password}
              testID="delete-confirm"
              onPress={() =>
                Alert.alert("Dernière confirmation", "Cette action est irréversible.", [
                  { text: "Annuler", style: "cancel" },
                  { text: "Supprimer", style: "destructive", onPress: () => void remove() }
                ])
              }
            />
            <Button label="Annuler" variant="ghost" onPress={() => setConfirming(false)} />
          </>
        ) : (
          <Button
            label="Supprimer mon compte"
            variant="danger"
            onPress={() => setConfirming(true)}
            testID="delete-start"
          />
        )}
      </View>
    </Screen>
  );
}
