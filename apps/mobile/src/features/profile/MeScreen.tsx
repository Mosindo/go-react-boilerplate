import React, { useState } from "react";
import { Alert, Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { deleteAccount } from "../../api/auth";
import type { MainStackParamList, MainTabParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Button, Input, Screen, Text, useToast } from "../../ui";
import { useAuth } from "../auth/AuthProvider";
import { useMyProfile } from "./hooks";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Me">,
  NativeStackScreenProps<MainStackParamList>
>;

function Row({ label, onPress }: { label: string; onPress: () => void }) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.row, { borderBottomColor: colors.border }]}
    >
      <Text>{label}</Text>
      <Text tone="muted">›</Text>
    </Pressable>
  );
}

export function MeScreen({ navigation }: Props) {
  const { signOut, user } = useAuth();
  const toast = useToast();
  const profile = useMyProfile().data?.profile;
  const [confirming, setConfirming] = useState(false);
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const remove = async () => {
    if (!password) {
      setError("Enter your password to confirm.");
      return;
    }
    setBusy(true);
    try {
      await deleteAccount(password);
      toast.show("Your account was deleted.");
      await signOut();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not delete your account.");
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <View style={styles.header}>
        <Avatar url={profile?.photos[0]?.url ?? null} name={profile?.firstName ?? "?"} size={72} />
        <View style={{ marginLeft: spacing.lg, flex: 1 }}>
          <Text variant="heading">{profile ? `${profile.firstName}, ${profile.age}` : "Your profile"}</Text>
          <Text tone="muted">{user?.email}</Text>
        </View>
      </View>
      <Row label="Edit profile" onPress={() => navigation.navigate("EditProfile")} />
      <Row label="Photos" onPress={() => navigation.navigate("Photos")} />
      <Row label="Discovery & privacy" onPress={() => navigation.navigate("Preferences")} />
      <Row label="Blocked people" onPress={() => navigation.navigate("Blocked")} />
      <Row label="Notifications" onPress={() => navigation.navigate("Notifications")} />
      <View style={{ height: spacing.xl }} />
      <Button label="Sign out" variant="secondary" onPress={() => void signOut()} testID="sign-out" />
      <View style={{ height: spacing.md }} />
      {confirming ? (
        <View>
          <Text tone="muted" style={{ marginBottom: spacing.md }}>
            This permanently deletes your profile, photos, matches and messages. It cannot be undone.
          </Text>
          <Input
            label="Password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoCapitalize="none"
            error={error}
          />
          <Button
            label="Delete my account forever"
            variant="danger"
            loading={busy}
            onPress={() =>
              Alert.alert("Delete account?", "All your data will be erased.", [
                { text: "Cancel", style: "cancel" },
                { text: "Delete", style: "destructive", onPress: () => void remove() }
              ])
            }
          />
          <Button
            label="Cancel"
            variant="ghost"
            onPress={() => {
              setConfirming(false);
              setPassword("");
              setError(null);
            }}
          />
        </View>
      ) : (
        <Button label="Delete account" variant="danger" onPress={() => setConfirming(true)} />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", marginBottom: spacing.xl },
  row: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth
  }
});
