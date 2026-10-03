import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { errorMessage, getOwnProfile, queryKeys } from "../api/platform";
import { Chip } from "../components/Chip";
import { PhotoImage } from "../components/PhotoImage";
import { Screen } from "../components/Screen";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Badge, Button, Card, Text, colors, radii, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

function MenuRow({ label, hint, onPress, testID }: { label: string; hint: string; onPress: () => void; testID: string }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.menuRow} testID={testID}>
      <View style={styles.menuCopy}>
        <Text weight="semibold">{label}</Text>
        <Text tone="muted" variant="caption">
          {hint}
        </Text>
      </View>
      <Text tone="muted" weight="bold">
        ›
      </Text>
    </Pressable>
  );
}

export default function ProfileScreen() {
  const navigation = useNavigation<Nav>();
  const query = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const profile = query.data;

  if (query.isLoading) {
    return <LoadingView fullScreen label="Loading your profile…" />;
  }
  if (query.isError || !profile) {
    return <ErrorView message={errorMessage(query.error, "Profile not found.")} onAction={() => void query.refetch()} />;
  }

  return (
    <Screen edges={["top", "left", "right"]} onRefresh={() => void query.refetch()} refreshing={query.isRefetching} testID="profile-screen">
      <View style={styles.hero}>
        <PhotoImage fallbackLabel={profile.firstName} path={profile.photos[0]?.url} style={styles.avatar} />
        <View style={styles.heroCopy}>
          <Text variant="title" weight="bold">
            {profile.firstName}, {profile.age}
          </Text>
          <Text tone="muted">{profile.city || "No city set"}</Text>
          <Badge label={profile.isVisible ? "Visible in discovery" : "Hidden from discovery"} variant={profile.isVisible ? "success" : "muted"} />
        </View>
      </View>
      {profile.bio ? <Text>{profile.bio}</Text> : null}
      {profile.interests.length > 0 ? (
        <View style={styles.wrap}>
          {profile.interests.map((i) => (
            <Chip key={i.slug} label={i.label} />
          ))}
        </View>
      ) : null}
      <Button label="Preview as others see me" onPress={() => navigation.navigate("ProfileDetail", { userId: profile.id, name: profile.firstName })} size="sm" variant="secondary" />

      <Card padding="sm" style={styles.menu}>
        <MenuRow hint={`${profile.photos.length} of 6 photos`} label="Photos" onPress={() => navigation.navigate("Photos")} testID="menu-photos" />
        <MenuRow hint="Name, bio, interests, location" label="Edit profile" onPress={() => navigation.navigate("EditProfile")} testID="menu-edit" />
        <MenuRow hint="Who you see, ages and distance" label="Discovery preferences" onPress={() => navigation.navigate("Preferences")} testID="menu-preferences" />
        <MenuRow hint="Visibility, blocked people, account" label="Privacy & account" onPress={() => navigation.navigate("Settings")} testID="menu-settings" />
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  avatar: { width: 96, height: 96, borderRadius: 48, borderWidth: 3, borderColor: colors.primaryBorder },
  heroCopy: { flex: 1, gap: spacing.xs },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  menu: { gap: 0 },
  menuRow: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.md, borderRadius: radii.md },
  menuCopy: { flex: 1, gap: 2 }
});
