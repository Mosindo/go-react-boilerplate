import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useNotifications, useOwnProfile } from "../../hooks/queries";
import type { MainStackParams, TabParams } from "../../navigation/types";
import { ErrorState, Loading, PhotoImage, Screen, Text } from "../../ui";
import { MIN_TOUCH, radius, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Me">,
  NativeStackScreenProps<MainStackParams>
>;

export default function MeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const profile = useOwnProfile();
  const notifications = useNotifications();

  if (profile.isPending) return <Loading />;
  if (profile.isError || !profile.data)
    return <ErrorState error={profile.error} onRetry={() => void profile.refetch()} />;
  const me = profile.data;
  const unread = notifications.data?.pages[0]?.unreadCount ?? 0;

  const rows: { label: string; hint?: string; badge?: number; onPress: () => void; testID: string }[] = [
    {
      label: "Modifier mon profil",
      hint: "Prénom, bio, centres d'intérêt, ville",
      onPress: () => navigation.navigate("EditProfile"),
      testID: "me-edit"
    },
    {
      label: "Mes photos",
      hint: `${me.photos.length}/6`,
      onPress: () => navigation.navigate("Photos"),
      testID: "me-photos"
    },
    {
      label: "Préférences de rencontre",
      hint: "Genre, âge, distance",
      onPress: () => navigation.navigate("Preferences"),
      testID: "me-prefs"
    },
    {
      label: "Notifications",
      ...(unread > 0 ? { badge: unread } : {}),
      onPress: () => navigation.navigate("Notifications"),
      testID: "me-notifications"
    },
    {
      label: "Confidentialité et compte",
      hint: "Visibilité, blocages, suppression",
      onPress: () => navigation.navigate("Settings"),
      testID: "me-settings"
    }
  ];

  return (
    <Screen scroll padded={false} testID="me-screen">
      <View style={styles.hero}>
        <PhotoImage photoId={me.photos[0]?.id} name={me.firstName} style={styles.avatar} />
        <Text variant="title">
          {me.firstName}, {me.age}
        </Text>
        {!me.isVisible ? (
          <Text variant="caption" tone="muted">
            Votre profil est masqué
          </Text>
        ) : null}
      </View>
      <View style={styles.list}>
        {rows.map((row) => (
          <Pressable
            key={row.label}
            accessibilityRole="button"
            accessibilityLabel={row.label}
            onPress={row.onPress}
            testID={row.testID}
            style={({ pressed }) => [
              styles.row,
              { backgroundColor: colors.surface, borderColor: colors.border, opacity: pressed ? 0.85 : 1 }
            ]}
          >
            <View style={styles.flex}>
              <Text bold>{row.label}</Text>
              {row.hint ? (
                <Text variant="caption" tone="muted">
                  {row.hint}
                </Text>
              ) : null}
            </View>
            {row.badge ? (
              <View style={[styles.badge, { backgroundColor: colors.accent }]}>
                <Text variant="caption" tone="onAccent" bold>
                  {row.badge}
                </Text>
              </View>
            ) : null}
            <Text tone="muted">›</Text>
          </Pressable>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.sm, paddingVertical: spacing.xl },
  avatar: { width: 112, height: 112, borderRadius: 56, overflow: "hidden" },
  list: { padding: spacing.lg, gap: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    minHeight: MIN_TOUCH + 12,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.lg
  },
  flex: { flex: 1 },
  badge: {
    minWidth: 24,
    height: 24,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center",
    paddingHorizontal: 6
  }
});
