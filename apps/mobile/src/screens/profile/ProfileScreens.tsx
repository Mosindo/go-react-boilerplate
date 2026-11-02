import React, { useState } from "react";
import { Switch, Text, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiErrorMessage } from "../../api/client";
import { profileApi, safetyApi } from "../../api/endpoints";
import type { Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";
import { PhotoManager } from "../../components/PhotoManager";
import {
  AboutFields,
  DEFAULT_PREFERENCES,
  IdentityFields,
  LocationFields,
  PreferenceFields,
  buildProfileInput,
  identityFromProfile,
  validateIdentity,
  type IdentityDraft,
  type LocationDraft
} from "../../components/ProfileForm";
import { useToast } from "../../components/Toast";
import { AppText, Avatar, Button, EmptyState, ErrorState, Loading, Row, Screen, TextField } from "../../components/ui";
import { PROFILE_KEY, useProfileBundle } from "../../hooks";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme/theme";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function MeScreen() {
  const navigation = useNavigation<Nav>();
  const { logout, user } = useAuth();
  const bundle = useProfileBundle();
  const profile = bundle.data?.profile;
  if (bundle.isLoading) return <Loading />;
  if (bundle.isError || !profile) return <ErrorState message={apiErrorMessage(bundle.error)} onRetry={() => void bundle.refetch()} />;
  return (
    <Screen scroll edges={["top"]}>
      <View style={{ alignItems: "center", gap: spacing.sm, paddingVertical: spacing.lg }}>
        <Avatar url={profile.photos[0]?.url} name={profile.firstName} size={104} />
        <AppText variant="title">
          {profile.firstName}, {profile.age}
        </AppText>
        <AppText muted>{user?.email}</AppText>
        {!profile.discoverable ? <AppText color="#E9A23B">⏸ Profil en pause : vous n'apparaissez plus dans la découverte.</AppText> : null}
      </View>
      <Row
        title="Modifier mon profil"
        subtitle="Prénom, bio, centres d'intérêt, position"
        onPress={() => navigation.navigate("EditProfile")}
      />
      <Row title="Mes photos" subtitle={`${profile.photos.length}/6`} onPress={() => navigation.navigate("Photos")} />
      <Row title="Préférences de rencontre" subtitle="Genre, âge, distance" onPress={() => navigation.navigate("Preferences")} />
      <Row title="Confidentialité et compte" subtitle="Pause, distance, suppression" onPress={() => navigation.navigate("Privacy")} />
      <Row title="Personnes bloquées" onPress={() => navigation.navigate("Blocked")} />
      <View style={{ height: spacing.lg }} />
      <Button label="Se déconnecter" variant="secondary" onPress={() => void logout()} testID="logout" />
      <AppText variant="caption" muted center>
        Tandem est gratuit : aucune fonctionnalité n'est payante ni limitée.
      </AppText>
    </Screen>
  );
}

export function EditProfileScreen() {
  const navigation = useNavigation<Nav>();
  const bundle = useProfileBundle();
  if (bundle.isLoading) return <Loading />;
  const profile = bundle.data?.profile;
  if (!profile) return <ErrorState message="Profil introuvable." onRetry={() => void bundle.refetch()} />;
  return <EditProfileForm onDone={() => navigation.goBack()} />;
}

function EditProfileForm({ onDone }: { onDone: () => void }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const profile = useProfileBundle().data?.profile ?? null;
  const [identity, setIdentity] = useState<IdentityDraft>(identityFromProfile(profile));
  const [location, setLocation] = useState<LocationDraft>({
    latitude: profile?.latitude ?? null,
    longitude: profile?.longitude ?? null,
    city: profile?.city ?? ""
  });
  const [bio, setBio] = useState(profile?.bio ?? "");
  const [interestIds, setInterestIds] = useState(profile?.interests.map((i) => i.id) ?? []);
  const [errors, setErrors] = useState<Partial<Record<"firstName" | "birthDate" | "gender", string>>>({});

  const save = useMutation({
    mutationFn: async () => {
      const input = buildProfileInput(identity, location, bio, interestIds, {
        showDistance: profile?.showDistance ?? true,
        discoverable: profile?.discoverable ?? true
      });
      if (!input) throw new Error("Profil invalide.");
      return profileApi.save(input);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(PROFILE_KEY, data);
      void queryClient.invalidateQueries({ queryKey: ["user"] });
      toast.show("Profil mis à jour.", "success");
      onDone();
    },
    onError: (e) => toast.show(apiErrorMessage(e), "error")
  });

  const submit = () => {
    const found = validateIdentity(identity);
    setErrors(found);
    if (Object.keys(found).length === 0) save.mutate();
  };

  return (
    <Screen scroll edges={["bottom"]}>
      <IdentityFields draft={identity} onChange={setIdentity} errors={errors} />
      <AboutFields bio={bio} onBio={setBio} selected={interestIds} onSelected={setInterestIds} />
      <AppText variant="heading">Position</AppText>
      <LocationFields value={location} onChange={setLocation} />
      <Button label="Enregistrer" onPress={submit} loading={save.isPending} testID="save-profile" />
    </Screen>
  );
}

export function PreferencesScreen() {
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const queryClient = useQueryClient();
  const bundle = useProfileBundle();
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const value = prefs ?? bundle.data?.preferences ?? DEFAULT_PREFERENCES;

  const save = useMutation({
    mutationFn: () => profileApi.savePreferences(value),
    onSuccess: (data) => {
      queryClient.setQueryData(PROFILE_KEY, data);
      toast.show("Préférences enregistrées.", "success");
      navigation.goBack();
    },
    onError: (e) => toast.show(apiErrorMessage(e), "error")
  });

  if (bundle.isLoading) return <Loading />;
  return (
    <Screen scroll edges={["bottom"]}>
      <PreferenceFields value={value} onChange={setPrefs} />
      <Button label="Enregistrer" onPress={() => save.mutate()} loading={save.isPending} testID="save-preferences" />
    </Screen>
  );
}

export function PhotosScreen() {
  return (
    <Screen scroll edges={["bottom"]}>
      <PhotoManager />
    </Screen>
  );
}

export function PrivacyScreen() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const { deleteAccount } = useAuth();
  const bundle = useProfileBundle();
  const profile = bundle.data?.profile;
  const [confirming, setConfirming] = useState(false);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = useMutation({
    mutationFn: async (patch: { showDistance?: boolean; discoverable?: boolean }) => {
      if (!profile) throw new Error("Profil introuvable.");
      const input = buildProfileInput(
        identityFromProfile(profile),
        { latitude: profile.latitude, longitude: profile.longitude, city: profile.city },
        profile.bio,
        profile.interests.map((i) => i.id),
        {
          showDistance: patch.showDistance ?? profile.showDistance,
          discoverable: patch.discoverable ?? profile.discoverable
        }
      );
      if (!input) throw new Error("Profil invalide.");
      return profileApi.save(input);
    },
    onSuccess: (data) => queryClient.setQueryData(PROFILE_KEY, data),
    onError: (e) => toast.show(apiErrorMessage(e), "error")
  });

  const remove = async () => {
    if (!password) return setError("Saisissez votre mot de passe.");
    setBusy(true);
    setError(null);
    try {
      await deleteAccount(password);
    } catch (e) {
      setError(apiErrorMessage(e));
      setBusy(false);
    }
  };

  if (!profile) return <Loading />;
  return (
    <Screen scroll edges={["bottom"]}>
      <Row
        title="Mettre mon profil en pause"
        subtitle="Vous n'êtes plus proposé·e à d'autres personnes. Vos matchs et conversations restent actifs."
        right={
          <Switch
            accessibilityLabel="Mettre mon profil en pause"
            value={!profile.discoverable}
            onValueChange={(paused) => update.mutate({ discoverable: !paused })}
          />
        }
      />
      <Row
        title="Afficher ma distance"
        subtitle="Une distance approximative (par tranche de 5 km) est montrée aux autres. Jamais votre position exacte."
        right={
          <Switch
            accessibilityLabel="Afficher ma distance"
            value={profile.showDistance}
            onValueChange={(show) => update.mutate({ showDistance: show })}
          />
        }
      />
      <View style={{ height: spacing.xl }} />
      <AppText variant="heading">Supprimer mon compte</AppText>
      <AppText muted>
        Votre profil, vos photos, vos likes, vos matchs et tous vos messages sont définitivement supprimés. Cette action est irréversible.
      </AppText>
      {!confirming ? (
        <Button label="Supprimer mon compte" variant="danger" onPress={() => setConfirming(true)} testID="delete-account" />
      ) : (
        <View style={{ gap: spacing.md }}>
          <TextField
            label="Confirmez avec votre mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            error={error}
            testID="delete-password"
          />
          <Button label="Supprimer définitivement" variant="danger" onPress={() => void remove()} loading={busy} testID="delete-confirm" />
          <Button
            label="Annuler"
            variant="ghost"
            onPress={() => {
              setConfirming(false);
              setPassword("");
              setError(null);
            }}
          />
        </View>
      )}
    </Screen>
  );
}

export function BlockedScreen() {
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useQuery({ queryKey: ["blocked"], queryFn: safetyApi.blocked });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["blocked"] });
      toast.show("Utilisateur débloqué.", "success");
    },
    onError: (e) => toast.show(apiErrorMessage(e), "error")
  });
  if (query.isLoading) return <Loading />;
  if (query.isError) return <ErrorState message={apiErrorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const items = query.data?.items ?? [];
  if (items.length === 0)
    return <EmptyState emoji="🛡️" title="Personne n'est bloqué" message="Les personnes que vous bloquez apparaîtront ici." />;
  return (
    <Screen scroll edges={["bottom"]}>
      {items.map((b) => (
        <Row
          key={b.userId}
          title={b.firstName || "Utilisateur"}
          right={
            <Text onPress={() => unblock.mutate(b.userId)} accessibilityRole="button" style={{ color: "#D9482B", fontWeight: "700" }}>
              Débloquer
            </Text>
          }
        />
      ))}
    </Screen>
  );
}
