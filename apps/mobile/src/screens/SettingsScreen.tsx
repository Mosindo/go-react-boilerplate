import React, { useState } from "react";
import { Pressable, StyleSheet, Switch, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authApi, profileApi, safetyApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthContext";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useTheme } from "../theme";
import { Button, Dialog, EmptyState, Input, LoadingState, Screen, Text, useToast } from "../ui";

type Props = NativeStackScreenProps<MainStackParams, "Settings">;

function ToggleRow({
  label,
  description,
  value,
  onChange
}: {
  label: string;
  description: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  const t = useTheme();
  return (
    <View style={styles.toggle}>
      <View style={styles.flex}>
        <Text variant="label">{label}</Text>
        <Text variant="caption" tone="muted">
          {description}
        </Text>
      </View>
      <Switch
        accessibilityLabel={label}
        value={value}
        onValueChange={onChange}
        trackColor={{ true: t.primary, false: t.border }}
      />
    </View>
  );
}

export function SettingsScreen({ navigation }: Props) {
  const toast = useToast();
  const qc = useQueryClient();
  const { user, signOut, deleteAccount } = useAuth();
  const overview = useQuery({ queryKey: queryKeys.overview, queryFn: profileApi.overview });
  const privacy = overview.data?.privacy;
  const [dialog, setDialog] = useState<"password" | "delete" | null>(null);
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const close = () => {
    setDialog(null);
    setCurrent("");
    setNext("");
  };

  const savePrivacy = useMutation({
    mutationFn: profileApi.savePrivacy,
    onSuccess: (o) => {
      qc.setQueryData(queryKeys.overview, o);
      qc.removeQueries({ queryKey: queryKeys.feed });
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  const changePassword = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      toast.show("Mot de passe modifié. Reconnectez-vous.");
      close();
      void signOut();
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  const remove = useMutation({
    mutationFn: () => deleteAccount(current),
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  if (overview.isLoading) return <LoadingState />;

  return (
    <Screen scroll>
      <Text variant="heading">Confidentialité</Text>
      {privacy ? (
        <>
          <ToggleRow
            label="Profil visible"
            description="Désactivez pour faire une pause : vos matchs restent actifs mais vous n'apparaissez plus aux autres."
            value={privacy.isDiscoverable}
            onChange={(v) => savePrivacy.mutate({ ...privacy, isDiscoverable: v })}
          />
          <ToggleRow
            label="Afficher ma distance"
            description="Les autres voient une distance approximative, jamais votre position."
            value={privacy.showDistance}
            onChange={(v) => savePrivacy.mutate({ ...privacy, showDistance: v })}
          />
        </>
      ) : null}
      <Button
        label="Personnes bloquées"
        variant="secondary"
        onPress={() => navigation.navigate("Blocked")}
      />

      <Text variant="heading" style={styles.section}>
        Compte
      </Text>
      <Text tone="muted">{user?.email}</Text>
      <Button
        label="Changer de mot de passe"
        variant="secondary"
        onPress={() => setDialog("password")}
      />
      <Button label="Se déconnecter" variant="secondary" onPress={() => void signOut()} />
      <Button label="Supprimer mon compte" variant="danger" onPress={() => setDialog("delete")} />

      <Dialog
        visible={dialog === "password"}
        title="Changer de mot de passe"
        confirmLabel="Modifier"
        loading={changePassword.isPending}
        onConfirm={() =>
          current && next.length >= 8
            ? changePassword.mutate()
            : toast.show("Mot de passe actuel requis et 8 caractères minimum.", "error")
        }
        onCancel={close}
      >
        <Input
          label="Mot de passe actuel"
          value={current}
          onChangeText={setCurrent}
          secureTextEntry
        />
        <Input
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureTextEntry
          maxLength={72}
        />
      </Dialog>
      <Dialog
        visible={dialog === "delete"}
        title="Supprimer définitivement ?"
        message="Votre profil, vos photos, vos matchs et vos messages seront effacés immédiatement. Cette action est irréversible."
        confirmLabel="Supprimer mon compte"
        confirmVariant="danger"
        loading={remove.isPending}
        onConfirm={() =>
          current ? remove.mutate() : toast.show("Saisissez votre mot de passe.", "error")
        }
        onCancel={close}
      >
        <Input
          label="Confirmez avec votre mot de passe"
          value={current}
          onChangeText={setCurrent}
          secureTextEntry
        />
      </Dialog>
    </Screen>
  );
}

export function BlockedScreen() {
  const t = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const q = useQuery({ queryKey: queryKeys.blocks, queryFn: safetyApi.blocks });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.blocks });
      qc.removeQueries({ queryKey: queryKeys.feed });
    },
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  if (q.isLoading) return <LoadingState />;
  const items = q.data ?? [];
  return (
    <Screen scroll>
      {items.length === 0 ? (
        <EmptyState icon="shield-checkmark-outline" title="Personne n'est bloqué" />
      ) : (
        items.map((b) => (
          <View key={b.id} style={[styles.blocked, { borderColor: t.border }]}>
            <Text variant="heading" style={styles.flex}>
              {b.firstName || "Utilisateur"}
            </Text>
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Débloquer ${b.firstName}`}
              onPress={() => unblock.mutate(b.id)}
            >
              <Text tone="primary" variant="label">
                Débloquer
              </Text>
            </Pressable>
          </View>
        ))
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  toggle: { flexDirection: "row", alignItems: "center", gap: 12 },
  section: { marginTop: 12 },
  blocked: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth
  }
});
