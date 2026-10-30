import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage, useProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { Avatar, ErrorView, LoadingView, Screen, Text, radii, spacing, useTheme } from "../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;
type Icon = React.ComponentProps<typeof Ionicons>["name"];

function Row({ icon, label, onPress, testID }: { icon: Icon; label: string; onPress: () => void; testID: string }) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.row, { backgroundColor: colors.surface, borderColor: colors.border }]}
      testID={testID}
    >
      <Ionicons color={colors.primary} name={icon} size={22} />
      <Text style={styles.rowLabel}>{label}</Text>
      <Ionicons color={colors.textMuted} name="chevron-forward" size={20} />
    </Pressable>
  );
}

export default function MeScreen() {
  const navigation = useNavigation<Nav>();
  const query = useProfile();

  if (query.isLoading) return <LoadingView />;
  if (query.isError || !query.data)
    return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  const profile = query.data;

  return (
    <Screen scroll testID="me-screen" topInset>
      <View style={styles.hero}>
        <Avatar name={profile.displayName} size={96} uri={profile.photos[0]?.url} />
        <Text variant="title" testID="me-name">
          {profile.displayName}, {profile.age}
        </Text>
        <Text tone="muted">{profile.city || "Position enregistrée"}</Text>
      </View>
      <View style={styles.list}>
        <Row
          icon="create-outline"
          label="Modifier mon profil"
          onPress={() => navigation.navigate("EditProfile")}
          testID="row-edit"
        />
        <Row
          icon="images-outline"
          label="Mes photos"
          onPress={() => navigation.navigate("Photos")}
          testID="row-photos"
        />
        <Row
          icon="options-outline"
          label="Préférences de rencontre"
          onPress={() => navigation.navigate("Preferences")}
          testID="row-preferences"
        />
        <Row
          icon="lock-closed-outline"
          label="Confidentialité et blocages"
          onPress={() => navigation.navigate("Privacy")}
          testID="row-privacy"
        />
        <Row
          icon="settings-outline"
          label="Compte et sécurité"
          onPress={() => navigation.navigate("Settings")}
          testID="row-settings"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.sm, marginVertical: spacing.xl },
  list: { gap: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    minHeight: 56,
    paddingHorizontal: spacing.lg,
    borderRadius: radii.md,
    borderWidth: 1
  },
  rowLabel: { flex: 1 }
});
