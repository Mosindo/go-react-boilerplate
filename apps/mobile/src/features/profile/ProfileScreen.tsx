import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useAuth } from "../../hooks/useAuth";
import { useOwnProfile } from "../../hooks/useProfile";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useColors } from "../../theme";
import {
  Card,
  ConfirmDialog,
  ErrorState,
  LoadingState,
  NavRow,
  ProfilePhoto,
  Screen,
  SectionTitle,
  Text
} from "../../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function ProfileScreen() {
  const navigation = useNavigation<Nav>();
  const colors = useColors();
  const { logout, user } = useAuth();
  const profile = useOwnProfile();
  const [confirmLogout, setConfirmLogout] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);

  if (profile.isPending) return <LoadingState />;
  if (profile.isError || !profile.data)
    return <ErrorState error={profile.error} onRetry={() => void profile.refetch()} />;
  const me = profile.data;

  return (
    <SafeAreaView edges={["top", "left", "right"]} style={[styles.screen, { backgroundColor: colors.background }]}>
      <Screen edges={[]} scroll>
        <View style={styles.hero}>
          <ProfilePhoto name={me.displayName} photo={me.photos[0]} round size="full" style={styles.avatar} />
          <Text variant="heading">
            {me.displayName}, {me.age}
          </Text>
          <Text tone="muted">{user?.email}</Text>
          {!me.discoverable ? (
            <Text tone="muted" variant="small">
              Your profile is hidden from Discover.
            </Text>
          ) : null}
        </View>

        <SectionTitle>Profile</SectionTitle>
        <Card>
          <NavRow icon="create-outline" label="Edit profile" onPress={() => navigation.navigate("EditProfile")} />
          <NavRow
            icon="images-outline"
            label="Photos"
            detail={`${me.photos.length}/6`}
            onPress={() => navigation.navigate("ManagePhotos")}
          />
          <NavRow
            icon="options-outline"
            label="Discovery preferences"
            onPress={() => navigation.navigate("Preferences")}
          />
        </Card>

        <SectionTitle>Safety and privacy</SectionTitle>
        <Card>
          <NavRow
            icon="shield-checkmark-outline"
            label="Safety tips"
            onPress={() => navigation.navigate("SafetyTips")}
          />
          <NavRow icon="ban-outline" label="Blocked people" onPress={() => navigation.navigate("BlockedUsers")} />
        </Card>

        <SectionTitle>Account</SectionTitle>
        <Card>
          <NavRow icon="key-outline" label="Change password" onPress={() => navigation.navigate("ChangePassword")} />
          <NavRow icon="log-out-outline" label="Log out" onPress={() => setConfirmLogout(true)} />
          <NavRow
            destructive
            icon="trash-outline"
            label="Delete account"
            onPress={() => navigation.navigate("DeleteAccount")}
          />
        </Card>
        <Text center tone="muted" variant="caption">
          Lumen is free. No subscriptions, no limits.
        </Text>
      </Screen>
      <ConfirmDialog
        confirmLabel="Log out"
        loading={loggingOut}
        onCancel={() => setConfirmLogout(false)}
        onConfirm={async () => {
          setLoggingOut(true);
          await logout();
        }}
        title="Log out of Lumen?"
        visible={confirmLogout}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  hero: { alignItems: "center", gap: spacing.xxs, paddingVertical: spacing.sm },
  avatar: { width: 120, height: 120, marginBottom: spacing.xs }
});
