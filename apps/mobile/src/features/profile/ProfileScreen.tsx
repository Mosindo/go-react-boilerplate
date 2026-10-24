import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthProvider";
import type { MainStackParamList } from "../../navigation/types";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Avatar, ErrorState, Icon, LoadingState, Screen, Text, useDialog } from "../../ui";
import { useOwnProfile } from "./useOwnProfile";

type Route = Exclude<keyof MainStackParamList, "Tabs" | "ProfileDetail" | "Chat" | "MatchFound">;

const ROWS: {
  route: Route;
  label: string;
  icon: React.ComponentProps<typeof Icon>["name"];
  testID: string;
}[] = [
  {
    route: "EditProfile",
    label: "Modifier mon profil",
    icon: "person-outline",
    testID: "profile-edit",
  },
  { route: "ManagePhotos", label: "Mes photos", icon: "images-outline", testID: "profile-photos" },
  {
    route: "Preferences",
    label: "Mes préférences de rencontre",
    icon: "options-outline",
    testID: "profile-preferences",
  },
  {
    route: "Privacy",
    label: "Confidentialité",
    icon: "lock-closed-outline",
    testID: "profile-privacy",
  },
  {
    route: "BlockedUsers",
    label: "Personnes bloquées",
    icon: "ban-outline",
    testID: "profile-blocked",
  },
  {
    route: "Notifications",
    label: "Notifications",
    icon: "notifications-outline",
    testID: "profile-notifications",
  },
];

export function ProfileScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { logout, user } = useAuth();
  const dialog = useDialog();
  const { data: own, error, isLoading, refetch } = useOwnProfile();

  if (isLoading) return <LoadingState />;
  if (error || !own) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const confirmLogout = async () => {
    if (await dialog.confirm({ title: "Se déconnecter ?", confirmLabel: "Se déconnecter" }))
      await logout();
  };

  return (
    <Screen testID="profile-screen" scroll edges={["top"]}>
      <View style={styles.hero}>
        <Avatar photoId={own.photos[0]?.id} name={own.firstName} size={104} />
        <Text variant="title" testID="profile-name">
          {own.firstName}, {own.age}
        </Text>
        <Text muted>{user?.email}</Text>
      </View>
      <View style={[styles.group, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        {ROWS.map((row) => (
          <Pressable
            key={row.route}
            testID={row.testID}
            accessibilityRole="button"
            onPress={() => navigation.navigate(row.route)}
            style={[styles.row, { borderBottomColor: colors.border }]}
          >
            <Icon name={row.icon} />
            <Text style={styles.flex}>{row.label}</Text>
            <Icon name="chevron-forward" size={18} color={colors.textMuted} />
          </Pressable>
        ))}
      </View>
      <View style={[styles.group, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        <Pressable
          testID="profile-logout"
          accessibilityRole="button"
          onPress={confirmLogout}
          style={[styles.row, { borderBottomColor: colors.border }]}
        >
          <Icon name="log-out-outline" />
          <Text style={styles.flex}>Se déconnecter</Text>
        </Pressable>
        <Pressable
          testID="profile-delete"
          accessibilityRole="button"
          onPress={() => navigation.navigate("DeleteAccount")}
          style={[styles.row, styles.lastRow]}
        >
          <Icon name="trash-outline" color={colors.danger} />
          <Text color={colors.danger} style={styles.flex}>
            Supprimer mon compte
          </Text>
        </Pressable>
      </View>
      <Text variant="caption" muted center>
        Alba est 100 % gratuit : aucun abonnement, aucune limite, aucune publicité.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.sm, paddingVertical: spacing.lg },
  group: { borderWidth: 1, borderRadius: radius.lg, overflow: "hidden" },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    minHeight: 56,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  lastRow: { borderBottomWidth: 0 },
  flex: { flex: 1 },
});
