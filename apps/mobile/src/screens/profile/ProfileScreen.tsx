import React, { useState } from "react";
import { Alert, Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthContext";
import { AuthImage, Button, ErrorState, LoadingState, ProfileView, Screen, Text } from "../../components";
import { useOwnProfile } from "../../hooks/queries";
import { errorMessage } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { MIN_TOUCH, radii, spacing, useTheme } from "../../theme";

type Route = keyof MainStackParamList;

function Row({ label, onPress, danger, testID }: { label: string; onPress: () => void; danger?: boolean; testID?: string }) {
  const { colors } = useTheme();
  return (
    <Pressable accessibilityRole="button" onPress={onPress} testID={testID} style={[styles.row, { borderBottomColor: colors.border }]}>
      <Text style={{ flex: 1, color: danger ? colors.danger : colors.text }}>{label}</Text>
      <Text tone="muted">›</Text>
    </Pressable>
  );
}

export function ProfileScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { signOut } = useAuth();
  const query = useOwnProfile();
  const [preview, setPreview] = useState(false);
  const go = (route: Route) => () => navigation.navigate(route as "Photos");

  if (query.isPending) return <Screen edges={[]}><LoadingState /></Screen>;
  if (query.isError) return <Screen edges={[]}><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;
  const profile = query.data;

  return (
    <Screen padded={false} edges={[]} testID="profile-screen">
      <ScrollView contentContainerStyle={{ paddingBottom: spacing.xxl }}>
        {profile ? (
          <View style={styles.header}>
            <AuthImage url={profile.photos[0]?.url} style={styles.avatar} fallbackInitial={profile.firstName} />
            <Text variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            <Button title="Preview my profile" variant="secondary" onPress={() => setPreview(true)} testID="profile-preview" />
          </View>
        ) : null}
        <Row label="Edit profile" onPress={go("EditProfile")} testID="row-edit-profile" />
        <Row label="Photos" onPress={go("Photos")} testID="row-photos" />
        <Row label="Discovery preferences" onPress={go("Preferences")} testID="row-preferences" />
        <Row label="Privacy" onPress={go("Privacy")} testID="row-privacy" />
        <Row label="Blocked users" onPress={go("BlockedUsers")} testID="row-blocked" />
        <Row label="Change password" onPress={go("ChangePassword")} />
        <Row
          label="Sign out"
          testID="row-signout"
          onPress={() =>
            Alert.alert("Sign out?", undefined, [
              { text: "Cancel", style: "cancel" },
              { text: "Sign out", onPress: () => void signOut() }
            ])
          }
        />
        <Row label="Delete my account" danger onPress={go("DeleteAccount")} testID="row-delete-account" />
      </ScrollView>
      <Modal visible={preview} animationType="slide" onRequestClose={() => setPreview(false)}>
        <Screen padded={false} edges={["top", "bottom"]}>
          <ScrollView>{profile ? <ProfileView profile={profile} /> : null}</ScrollView>
          <View style={{ padding: spacing.lg, backgroundColor: colors.background }}>
            <Button title="Close" onPress={() => setPreview(false)} />
          </View>
        </Screen>
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { alignItems: "center", gap: spacing.md, padding: spacing.xl },
  avatar: { width: 120, height: 120, borderRadius: 60 },
  row: { minHeight: MIN_TOUCH + 8, flexDirection: "row", alignItems: "center", paddingHorizontal: spacing.lg, borderBottomWidth: StyleSheet.hairlineWidth, borderRadius: radii.sm }
});
