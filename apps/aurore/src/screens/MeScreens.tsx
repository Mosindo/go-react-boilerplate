import React, { useState } from "react";
import { Pressable, ScrollView, Switch, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authApi, moderationApi, profileApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { SelfProfile } from "../api/types";
import { useAuth } from "../auth/AuthProvider";
import { Avatar } from "../components/AuthImage";
import { useFeedback } from "../components/Feedback";
import { BasicsForm, InterestPicker, PhotoManager, PreferencesForm, ShareLocationButton, errorMessage } from "../components/forms";
import { Button, Card, EmptyState, ErrorState, Loading, Screen, Separator, Text, TextField } from "../components/ui";
import { MAX_BIO } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme/theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function useProfile() {
  return useQuery({ queryKey: keys.profile, queryFn: profileApi.get });
}

function Menu({ icon, label, onPress, testID }: { icon: keyof typeof Ionicons.glyphMap; label: string; onPress: () => void; testID?: string }) {
  const t = useTheme();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} testID={testID} onPress={onPress} style={{ flexDirection: "row", alignItems: "center", paddingVertical: spacing.lg, gap: spacing.md }}>
      <Ionicons name={icon} size={22} color={t.accent} />
      <Text style={{ flex: 1 }}>{label}</Text>
      <Ionicons name="chevron-forward" size={18} color={t.textMuted} />
    </Pressable>
  );
}

export function MeScreen() {
  const nav = useNavigation<Nav>();
  const { account } = useAuth();
  const profile = useProfile();
  if (profile.isLoading) return <Screen><Loading /></Screen>;
  if (profile.error || !profile.data) return <Screen><ErrorState message={(profile.error as Error | null)?.message ?? "Profil indisponible."} onRetry={() => void profile.refetch()} /></Screen>;
  const p = profile.data;
  return (
    <Screen scroll>
      <Text variant="title" style={{ paddingTop: 8, paddingBottom: spacing.lg }}>Mon profil</Text>
      <Card style={{ flexDirection: "row", alignItems: "center", gap: spacing.lg, marginBottom: spacing.lg }}>
        <Avatar path={p.photos[0]?.thumbUrl} size={72} />
        <View style={{ flex: 1 }}>
          <Text variant="heading">{p.firstName}, {p.age}</Text>
          <Text muted numberOfLines={1}>{account?.email}</Text>
          {!p.isVisible ? <Text variant="caption" muted>Profil en pause</Text> : null}
        </View>
      </Card>
      <Card>
        <Menu icon="create-outline" label="Modifier mon profil" onPress={() => nav.navigate("EditProfile")} testID="menu-edit" />
        <Separator />
        <Menu icon="images-outline" label="Mes photos" onPress={() => nav.navigate("Photos")} testID="menu-photos" />
        <Separator />
        <Menu icon="options-outline" label="Mes préférences de rencontre" onPress={() => nav.navigate("Preferences")} testID="menu-prefs" />
        <Separator />
        <Menu icon="shield-checkmark-outline" label="Confidentialité et compte" onPress={() => nav.navigate("Settings")} testID="menu-settings" />
      </Card>
    </Screen>
  );
}

export function EditProfileScreen({ navigation }: { navigation: Nav }) {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const profile = useProfile();
  const [bio, setBio] = useState<string | null>(null);
  const [interests, setInterests] = useState<number[] | null>(null);

  const save = useMutation({
    mutationFn: async (p: SelfProfile) => {
      await profileApi.saveInterests(interests ?? p.interests.map((i) => i.id));
      return profileApi.save({ firstName: p.firstName, birthDate: p.birthDate, gender: p.gender as "man" | "woman" | "non_binary", bio: (bio ?? p.bio).trim(), city: p.city });
    },
    onSuccess: (p) => {
      qc.setQueryData(keys.profile, p);
      toast("Profil mis à jour.", "success");
      navigation.goBack();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (!profile.data) return <Screen><Loading /></Screen>;
  const p = profile.data;
  const bioValue = bio ?? p.bio;
  const selected = interests ?? p.interests.map((i) => i.id);

  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={{ height: spacing.lg }} />
      <BasicsForm profile={p} submitLabel="Enregistrer l'identité" onSaved={() => toast("Informations enregistrées.", "success")} />
      <View style={{ height: spacing.xxl }} />
      <TextField label="À propos de moi" value={bioValue} onChangeText={setBio} multiline maxLength={MAX_BIO} hint={`${bioValue.length}/${MAX_BIO}`} testID="edit-bio" />
      <Text variant="label" style={{ marginBottom: spacing.sm }}>Centres d&apos;intérêt</Text>
      <InterestPicker selected={selected} onChange={setInterests} />
      <Button label="Enregistrer bio et centres d'intérêt" onPress={() => save.mutate(p)} loading={save.isPending} style={{ marginTop: spacing.xl }} testID="edit-save" />
      <View style={{ height: spacing.xl }} />
      <Text variant="label" style={{ marginBottom: spacing.sm }}>Localisation</Text>
      <Text variant="caption" muted style={{ marginBottom: spacing.md }}>
        {p.hasLocation ? `Position approximative enregistrée${p.city ? ` (${p.city})` : ""}.` : "Aucune position enregistrée."}
      </Text>
      <ShareLocationButton />
    </Screen>
  );
}

export function PhotosScreen() {
  const profile = useProfile();
  if (!profile.data) return <Screen><Loading /></Screen>;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={{ height: spacing.lg }} />
      <PhotoManager profile={profile.data} />
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: { navigation: Nav }) {
  const { toast } = useFeedback();
  const profile = useProfile();
  if (!profile.data) return <Screen><Loading /></Screen>;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={{ height: spacing.lg }} />
      <PreferencesForm
        profile={profile.data}
        submitLabel="Enregistrer"
        onSaved={() => {
          toast("Préférences enregistrées.", "success");
          navigation.goBack();
        }}
      />
    </Screen>
  );
}

function ToggleRow({ label, hint, value, onChange, testID }: { label: string; hint: string; value: boolean; onChange: (v: boolean) => void; testID?: string }) {
  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md, paddingVertical: spacing.md }}>
      <View style={{ flex: 1 }}>
        <Text variant="label">{label}</Text>
        <Text variant="caption" muted>{hint}</Text>
      </View>
      <Switch value={value} onValueChange={onChange} accessibilityLabel={label} testID={testID} />
    </View>
  );
}

export function SettingsScreen({ navigation }: { navigation: Nav }) {
  const qc = useQueryClient();
  const { toast, confirm } = useFeedback();
  const { signOut, forgetSession } = useAuth();
  const profile = useProfile();
  const [deleting, setDeleting] = useState(false);
  const [password, setPassword] = useState("");

  const update = useMutation({
    mutationFn: (patch: { isVisible?: boolean; showDistance?: boolean }) => {
      const p = profile.data as SelfProfile;
      return profileApi.save({ firstName: p.firstName, birthDate: p.birthDate, gender: p.gender as "man" | "woman" | "non_binary", bio: p.bio, city: p.city, isVisible: patch.isVisible ?? p.isVisible, showDistance: patch.showDistance ?? p.showDistance });
    },
    onSuccess: (p) => {
      qc.setQueryData(keys.profile, p);
      void qc.invalidateQueries({ queryKey: ["discover"] });
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      toast("Votre compte a été supprimé.", "success");
      await forgetSession();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  if (!profile.data) return <Screen><Loading /></Screen>;
  const p = profile.data;

  const confirmDelete = async () => {
    if (!password) return toast("Entrez votre mot de passe pour confirmer.", "error");
    const ok = await confirm({
      title: "Supprimer définitivement votre compte ?",
      message: "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.",
      confirmLabel: "Supprimer mon compte",
      destructive: true
    });
    if (ok) remove.mutate();
  };

  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={{ height: spacing.lg }} />
      <Card>
        <ToggleRow label="Profil visible" hint="Désactivez pour mettre votre profil en pause : personne ne vous verra, vos matchs restent." value={p.isVisible} onChange={(v) => update.mutate({ isVisible: v })} testID="toggle-visible" />
        <Separator />
        <ToggleRow label="Afficher ma distance" hint="Les autres voient une distance approximative, jamais votre position." value={p.showDistance} onChange={(v) => update.mutate({ showDistance: v })} testID="toggle-distance" />
      </Card>
      <View style={{ height: spacing.lg }} />
      <Card>
        <Menu icon="ban-outline" label="Personnes bloquées" onPress={() => navigation.navigate("Blocked")} testID="menu-blocked" />
      </Card>
      <View style={{ height: spacing.xl }} />
      <Button label="Se déconnecter" variant="secondary" onPress={() => void signOut()} testID="logout" />
      <View style={{ height: spacing.xxl }} />
      <Text variant="heading" style={{ marginBottom: spacing.sm }}>Supprimer mon compte</Text>
      {!deleting ? (
        <Button label="Supprimer mon compte…" variant="danger" onPress={() => setDeleting(true)} testID="delete-start" />
      ) : (
        <View>
          <TextField label="Mot de passe" value={password} onChangeText={setPassword} secureTextEntry testID="delete-password" />
          <Button label="Supprimer définitivement" variant="danger" onPress={() => void confirmDelete()} loading={remove.isPending} testID="delete-confirm" />
        </View>
      )}
    </Screen>
  );
}

export function BlockedScreen() {
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const query = useQuery({ queryKey: keys.blocked, queryFn: moderationApi.blocked });
  const unblock = useMutation({
    mutationFn: moderationApi.unblock,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.blocked });
      void qc.invalidateQueries({ queryKey: ["discover"] });
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  if (query.isLoading) return <Screen><Loading /></Screen>;
  if (query.error) return <Screen><ErrorState message={(query.error as Error).message} onRetry={() => void query.refetch()} /></Screen>;
  const items = query.data ?? [];
  if (items.length === 0) return <Screen><EmptyState icon="happy-outline" title="Personne n'est bloqué" message="Les personnes que vous bloquez apparaîtront ici." /></Screen>;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <View style={{ height: spacing.lg }} />
      <ScrollView scrollEnabled={false}>
        {items.map((b) => (
          <View key={b.userId} style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingVertical: spacing.md }}>
            <Text variant="heading">{b.firstName || "Ancien membre"}</Text>
            <Button label="Débloquer" variant="secondary" onPress={() => unblock.mutate(b.userId)} style={{ minHeight: 40 }} />
          </View>
        ))}
      </ScrollView>
    </Screen>
  );
}
