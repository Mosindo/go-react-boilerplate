import React, { useState } from "react";
import { FlatList, Pressable, StyleSheet, Text, View } from "react-native";
import * as Location from "expo-location";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import type { AppNotification, MyProfile } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { AboutFields, BasicsFields, PreferenceFields } from "../features/profile/forms";
import {
  draftFromProfile,
  hasErrors,
  toPreferences,
  toProfileInput,
  validateAbout,
  validateBasics,
  validatePreferences,
  type Draft,
  type Errors
} from "../features/profile/draft";
import { PhotoManager } from "../features/profile/PhotoManager";
import { confirmAction, notify } from "../lib/dialogs";
import { formatActivity } from "../lib/format";
import type { MainStackParams } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";
import { AuthImage } from "../ui/AuthImage";
import { Body, Button, EmptyState, ErrorState, ErrorText, Loading, Screen, TextField, Title, ToggleRow } from "../ui/components";

type Nav = NativeStackNavigationProp<MainStackParams>;

export function useMyProfile() {
  return useQuery({ queryKey: ["profile"], queryFn: api.myProfile, retry: false });
}

/** "Me" tab: summary + entry points. */
export function MeScreen() {
  const navigation = useNavigation<Nav>();
  const profile = useMyProfile();
  const theme = useTheme();
  if (profile.isLoading) {
    return <Loading />;
  }
  if (!profile.data) {
    return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  }
  const me = profile.data;
  return (
    <Screen>
      <View style={styles.center}>
        <AuthImage path={me.photos[0]?.url} label="Ma photo principale" style={styles.avatar} />
        <Title>{`${me.firstName}, ${me.age}`}</Title>
        {me.city ? <Body muted>{me.city}</Body> : null}
      </View>
      {me.bio ? <Body>{me.bio}</Body> : null}
      <Button title="Modifier mon profil" onPress={() => navigation.navigate("EditProfile")} testID="edit-profile" />
      <Button
        title="Réglages et confidentialité"
        variant="secondary"
        onPress={() => navigation.navigate("Settings")}
        testID="open-settings"
      />
      <Text style={{ color: theme.textMuted, textAlign: "center", marginTop: spacing.lg }}>
        Braise est 100 % gratuit : pas d&apos;abonnement, pas de limite de likes, pas de publicité.
      </Text>
    </Screen>
  );
}

export function EditProfileScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const [draft, setDraft] = useState<Draft | null>(null);
  const [errors, setErrors] = useState<Errors>({});
  const [error, setError] = useState<string | null>(null);

  const save = useMutation({
    mutationFn: async (value: Draft) => {
      await api.saveProfile(toProfileInput(value));
      await api.savePreferences(toPreferences(value));
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      navigation.goBack();
    },
    onError: (e) => setError(errorMessage(e))
  });

  if (!profile.data) {
    return <Loading />;
  }
  const current = draft ?? draftFromProfile(profile.data);
  const submit = () => {
    const found = { ...validateBasics(current), ...validateAbout(current), ...validatePreferences(current) };
    setErrors(found);
    if (!hasErrors(found)) {
      save.mutate(current);
    }
  };

  return (
    <Screen>
      <Title>Photos</Title>
      <PhotoManager photos={profile.data.photos} />
      <Title>À propos</Title>
      <BasicsFields draft={current} onChange={setDraft} errors={errors} />
      <AboutFields draft={current} onChange={setDraft} errors={errors} />
      <Title>Préférences</Title>
      <PreferenceFields draft={current} onChange={setDraft} errors={errors} />
      <ErrorText message={error} />
      <Button title="Enregistrer" onPress={submit} loading={save.isPending} testID="save-profile" />
    </Screen>
  );
}

export function SettingsScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const { signOut, afterAccountDeleted } = useAuth();
  const profile = useMyProfile();
  const [password, setPassword] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const privacy = useMutation({
    mutationFn: (value: Partial<MyProfile["privacy"]>) => api.savePrivacy(value),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["profile"] }),
    onError: (e) => setError(errorMessage(e))
  });

  const locate = useMutation({
    mutationFn: async () => {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        throw new Error("Autorisation de localisation refusée.");
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      await api.setLocation(position.coords.latitude, position.coords.longitude);
    },
    onSuccess: () => {
      notify("Position mise à jour", "Seule une position approximative est enregistrée.");
      return queryClient.invalidateQueries({ queryKey: ["profile"] });
    },
    onError: (e) => setError(errorMessage(e))
  });

  const clearLocation = useMutation({
    mutationFn: api.clearLocation,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["profile"] })
  });

  const remove = useMutation({
    mutationFn: () => api.deleteAccount(password),
    onSuccess: () => afterAccountDeleted(),
    onError: (e) => setError(errorMessage(e))
  });

  if (!profile.data) {
    return <Loading />;
  }
  const me = profile.data;

  return (
    <Screen>
      <Title>Confidentialité</Title>
      <ToggleRow
        label="Apparaître dans la découverte"
        description="Désactivé : votre profil n'est plus proposé, vos matchs restent actifs."
        value={me.privacy.discoverable}
        onChange={(discoverable) => privacy.mutate({ discoverable })}
      />
      <ToggleRow
        label="Afficher ma distance"
        description="Une distance approximative (par tranche de 5 km), jamais votre position exacte."
        value={me.privacy.showDistance}
        onChange={(showDistance) => privacy.mutate({ showDistance })}
      />
      <Button
        title={me.hasLocation ? "Actualiser ma position" : "Activer la localisation"}
        variant="secondary"
        onPress={() => locate.mutate()}
        loading={locate.isPending}
      />
      {me.hasLocation ? <Button title="Supprimer ma position" variant="ghost" onPress={() => clearLocation.mutate()} /> : null}

      <Title>Sécurité</Title>
      <Button title="Utilisateurs bloqués" variant="secondary" onPress={() => navigation.navigate("Blocked")} />

      <Title>Compte</Title>
      <Button title="Se déconnecter" variant="secondary" onPress={() => void signOut()} testID="logout" />
      {!confirming ? (
        <Button title="Supprimer mon compte" variant="danger" onPress={() => setConfirming(true)} testID="delete-account" />
      ) : (
        <View style={styles.danger}>
          <Body>Cette action est définitive : profil, photos, matchs et messages seront effacés.</Body>
          <TextField
            label="Confirmez avec votre mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            testID="delete-password"
          />
          <Button
            title="Supprimer définitivement"
            variant="danger"
            disabled={!password}
            loading={remove.isPending}
            testID="delete-confirm"
            onPress={() => confirmAction("Supprimer le compte ?", "Impossible d'annuler.", "Supprimer", () => remove.mutate())}
          />
          <Button
            title="Annuler"
            variant="ghost"
            onPress={() => {
              setConfirming(false);
              setPassword("");
            }}
          />
        </View>
      )}
      <ErrorText message={error} />
    </Screen>
  );
}

export function BlockedScreen() {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const blocks = useQuery({ queryKey: ["blocks"], queryFn: api.blocks });
  const unblock = useMutation({
    mutationFn: (id: string) => api.unblock(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["blocks"] })
  });
  if (blocks.isLoading) {
    return <Loading />;
  }
  if (blocks.isError) {
    return <ErrorState message={errorMessage(blocks.error)} onRetry={() => void blocks.refetch()} />;
  }
  if ((blocks.data ?? []).length === 0) {
    return <EmptyState title="Personne n'est bloqué" message="Les profils que vous bloquez apparaissent ici." />;
  }
  return (
    <FlatList
      data={blocks.data}
      keyExtractor={(b) => b.id}
      style={{ backgroundColor: theme.background }}
      renderItem={({ item }) => (
        <View style={[styles.row, { borderBottomColor: theme.border }]}>
          <AuthImage path={item.photoUrl} style={styles.small} />
          <Text style={{ flex: 1, color: theme.text, fontSize: 16, fontWeight: "600" }}>{item.firstName || "Compte"}</Text>
          <Button title="Débloquer" variant="secondary" onPress={() => unblock.mutate(item.id)} style={{ minHeight: 40 }} />
        </View>
      )}
    />
  );
}

export function NotificationsScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["notifications"], queryFn: api.notifications });
  const refresh = () => queryClient.invalidateQueries({ queryKey: ["notifications"] });
  const readAll = useMutation({ mutationFn: api.markAllNotificationsRead, onSuccess: refresh });
  const readOne = useMutation({ mutationFn: (id: string) => api.markNotificationRead(id), onSuccess: refresh });

  const open = (n: AppNotification) => {
    if (!n.readAt) {
      readOne.mutate(n.id);
    }
    navigation.navigate("Tabs", { screen: "Messages" } as never);
  };

  if (query.isLoading) {
    return <Loading />;
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }
  const items = query.data?.notifications ?? [];
  if (items.length === 0) {
    return <EmptyState title="Aucune notification" message="Vos nouveaux matchs apparaîtront ici." />;
  }
  return (
    <View style={[styles.flex, { backgroundColor: theme.background }]}>
      <Button
        title="Tout marquer comme lu"
        variant="ghost"
        onPress={() => readAll.mutate()}
        disabled={(query.data?.unreadCount ?? 0) === 0}
      />
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshing={query.isRefetching}
        onRefresh={() => void query.refetch()}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            onPress={() => open(item)}
            style={[styles.row, { borderBottomColor: theme.border, backgroundColor: item.readAt ? "transparent" : theme.surfaceAlt }]}
            testID="notification-item"
          >
            <AuthImage path={item.actor?.photoUrl} style={styles.small} />
            <View style={styles.flex}>
              <Text style={{ color: theme.text, fontWeight: item.readAt ? "500" : "800" }}>
                {item.actor ? `Nouveau match avec ${item.actor.firstName} !` : "Nouveau match !"}
              </Text>
              <Text style={{ color: theme.textMuted, fontSize: 12 }}>{formatActivity(item.createdAt)}</Text>
            </View>
          </Pressable>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  center: { alignItems: "center", gap: spacing.sm, marginVertical: spacing.lg },
  avatar: { width: 140, height: 140, borderRadius: 70 },
  small: { width: 48, height: 48, borderRadius: radius.pill },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.lg, borderBottomWidth: StyleSheet.hairlineWidth },
  danger: { gap: spacing.md }
});
