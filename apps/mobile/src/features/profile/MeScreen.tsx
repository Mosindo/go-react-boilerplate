import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../../auth/AuthContext";
import { useMyProfile } from "../../hooks/queries";
import type { RootParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { ErrorState, LoadingView } from "../../ui/Feedback";
import { Avatar } from "../../ui/Photo";
import { Screen } from "../../ui/Screen";
import { Text } from "../../ui/Text";

type Nav = NativeStackNavigationProp<RootParamList>;

function Row({
  icon,
  label,
  onPress,
  testID
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.row,
        { backgroundColor: pressed ? colors.surfaceMuted : colors.surface, borderColor: colors.border }
      ]}
    >
      <Ionicons name={icon} size={22} color={colors.primary} />
      <Text style={styles.flex}>{label}</Text>
      <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} />
    </Pressable>
  );
}

export function MeScreen() {
  const nav = useNavigation<Nav>();
  const { signOut } = useAuth();
  const { colors } = useTheme();
  const profile = useMyProfile();

  if (profile.isPending)
    return (
      <Screen edges={["top"]}>
        <LoadingView />
      </Screen>
    );
  if (profile.isError)
    return (
      <Screen edges={["top"]}>
        <ErrorState message={profile.error.message} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  const me = profile.data;

  return (
    <Screen scroll edges={["top"]}>
      <View style={styles.header}>
        <Avatar path={me.photos[0]?.url} name={me.firstName} size={112} />
        <Text variant="title" style={styles.name}>
          {me.firstName}, {me.age}
        </Text>
        {me.city ? <Text tone="textMuted">{me.city}</Text> : null}
        {!me.discoverable ? (
          <View style={[styles.paused, { backgroundColor: colors.primarySoft }]}>
            <Text variant="label" tone="primary">
              Profil en pause : tu n&apos;apparais pas dans la découverte
            </Text>
          </View>
        ) : null}
      </View>

      <Pressable
        onPress={() => nav.navigate("UserProfile", { userId: me.userId, card: me })}
        accessibilityRole="button"
        accessibilityLabel="Voir mon profil tel que les autres le voient"
      >
        <Text tone="primary" align="center" variant="label" style={styles.preview}>
          Voir mon profil tel que les autres le voient
        </Text>
      </Pressable>

      <View style={styles.list}>
        <Row
          icon="create-outline"
          label="Modifier mon profil et mes photos"
          onPress={() => nav.navigate("EditProfile")}
          testID="row-edit"
        />
        <Row
          icon="options-outline"
          label="Préférences de rencontre"
          onPress={() => nav.navigate("Preferences")}
          testID="row-prefs"
        />
        <Row
          icon="eye-off-outline"
          label="Confidentialité et blocages"
          onPress={() => nav.navigate("Privacy")}
          testID="row-privacy"
        />
        <Row
          icon="lock-closed-outline"
          label="Sécurité et compte"
          onPress={() => nav.navigate("Security")}
          testID="row-security"
        />
      </View>

      <Button title="Se déconnecter" variant="secondary" onPress={() => void signOut()} testID="logout" />
      <Text variant="caption" tone="textSubtle" align="center" style={styles.footer}>
        Ember est 100 % gratuit : aucun abonnement, aucun boost payant, aucune limite de likes.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { alignItems: "center", paddingTop: spacing.md },
  name: { marginTop: spacing.sm },
  paused: {
    marginTop: spacing.sm,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  preview: { marginVertical: spacing.md },
  list: { gap: spacing.xs, marginBottom: spacing.lg },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.md,
    borderRadius: radii.md,
    borderWidth: 1
  },
  footer: { marginTop: spacing.lg }
});
