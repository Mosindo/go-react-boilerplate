import React, { useState } from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import { errorMessage } from "../../api/client";
import { authApi, profileApi } from "../../api/endpoints";
import { keys, useOwnProfile } from "../../hooks/queries";
import { useSession } from "../../hooks/session";
import type { MainStackParams } from "../../navigation/types";
import { confirm } from "../../shared/feedback/confirm";
import { Screen } from "../../shared/layout";
import { Avatar, Button, ErrorState, Input, Loader, Sheet, Text, radius, spacing, useColors, useToast } from "../../shared/ui";
import { LocationStep } from "../onboarding/LocationStep";

function Row({ label, hint, onPress, testID }: { label: string; hint?: string; onPress: () => void; testID?: string }) {
  const c = useColors();
  return (
    <Pressable accessibilityRole="button" accessibilityLabel={label} onPress={onPress} testID={testID} style={[styles.row, { borderBottomColor: c.border }]}>
      <View style={{ flex: 1 }}>
        <Text variant="label">{label}</Text>
        {hint ? <Text variant="caption" tone="muted">{hint}</Text> : null}
      </View>
      <Text tone="muted">›</Text>
    </Pressable>
  );
}

export function MeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const toast = useToast();
  const c = useColors();
  const { signOut, dropSession } = useSession();
  const profile = useOwnProfile();
  const [locationSheet, setLocationSheet] = useState(false);
  const [deleteSheet, setDeleteSheet] = useState(false);
  const [password, setPassword] = useState("");

  const visibility = useMutation({
    mutationFn: (visible: boolean) => {
      const p = profile.data;
      if (!p) throw new Error("profil indisponible");
      return profileApi.save({ firstName: p.firstName, gender: p.gender, isVisible: visible });
    },
    onSuccess: (p) => qc.setQueryData(keys.profile, p),
    onError: (err) => toast.show(errorMessage(err), "error"),
  });

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: () => dropSession(),
    onError: (err) => toast.show(errorMessage(err), "error"),
  });

  if (profile.isPending) return <Loader />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  const p = profile.data;

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Avatar path={p.photos[0]?.url} name={p.firstName} size={96} />
        <Text variant="title">{p.firstName}, {p.age}</Text>
        {p.city ? <Text tone="muted">{p.city}</Text> : null}
      </View>

      <View style={[styles.switchRow, { backgroundColor: c.surface, borderColor: c.border }]}>
        <View style={{ flex: 1 }}>
          <Text variant="label">Masquer mon profil</Text>
          <Text variant="caption" tone="muted">Vous n'apparaissez plus dans la découverte. Vos conversations restent actives.</Text>
        </View>
        <Switch accessibilityLabel="Masquer mon profil" value={!p.isVisible} onValueChange={(hidden) => visibility.mutate(!hidden)} disabled={visibility.isPending} testID="hide-profile" />
      </View>

      <View>
        <Row label="Modifier mon profil" onPress={() => navigation.navigate("EditProfile")} testID="row-edit" />
        <Row label="Mes photos" hint={`${p.photos.length} photo${p.photos.length > 1 ? "s" : ""}`} onPress={() => navigation.navigate("Photos")} />
        <Row label="Mes préférences de rencontre" onPress={() => navigation.navigate("Preferences")} />
        <Row label="Ma position" hint={p.hasLocation ? "Enregistrée (arrondie à ~1 km)" : "Non partagée"} onPress={() => setLocationSheet(true)} />
        <Row label="Personnes bloquées" onPress={() => navigation.navigate("Blocked")} />
        <Row label="Changer le mot de passe" onPress={() => navigation.navigate("ChangePassword")} />
      </View>

      <Button title="Se déconnecter" variant="secondary" onPress={() => confirm("Se déconnecter ?", "Vous devrez vous reconnecter pour retrouver vos conversations.", "Se déconnecter", () => void signOut(), false)} testID="logout" />
      <Button title="Supprimer mon compte" variant="danger" onPress={() => setDeleteSheet(true)} testID="delete-account" />

      <Sheet visible={locationSheet} title="Ma position" onClose={() => setLocationSheet(false)}>
        <LocationStep hasLocation={p.hasLocation} initialCity={p.city} onDone={() => { setLocationSheet(false); void qc.invalidateQueries({ queryKey: keys.profile }); toast.show("Position mise à jour", "success"); }} />
      </Sheet>

      <Sheet visible={deleteSheet} title="Supprimer mon compte" onClose={() => setDeleteSheet(false)}>
        <Text tone="danger" variant="label">Cette action est définitive.</Text>
        <Text tone="muted">Votre profil, vos photos, vos matchs, vos conversations et vos données associées seront supprimés immédiatement et ne pourront pas être récupérés.</Text>
        <Input label="Confirmez avec votre mot de passe" value={password} onChangeText={setPassword} secureTextEntry testID="delete-password" />
        <Button title="Supprimer définitivement" variant="danger" onPress={() => remove.mutate()} disabled={password.length === 0} loading={remove.isPending} testID="delete-confirm" />
      </Sheet>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.sm, paddingVertical: spacing.md },
  switchRow: { flexDirection: "row", alignItems: "center", gap: spacing.md, borderWidth: 1, borderRadius: radius.md, padding: spacing.md },
  row: { flexDirection: "row", alignItems: "center", minHeight: 56, borderBottomWidth: StyleSheet.hairlineWidth, gap: spacing.md },
});
