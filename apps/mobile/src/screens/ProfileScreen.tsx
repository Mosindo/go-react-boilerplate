import React from "react";
import { Alert, StyleSheet, Switch, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { profileToInput } from "../domain/profileInput";
import { useAuth, useMe } from "../hooks/useAuth";
import { useSaveProfile } from "../hooks/useProfile";
import type { MainStackParamList, TabParamList } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Avatar, ListItem, Text } from "../shared/ui";
import { spacing, useTheme } from "../theme";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, "Profile">,
  NativeStackScreenProps<MainStackParamList>
>;

function Chevron() {
  return (
    <Text tone="muted" variant="heading">
      ›
    </Text>
  );
}

export default function ProfileScreen({ navigation }: Props) {
  const theme = useTheme();
  const meQuery = useMe();
  const { signOut } = useAuth();
  const save = useSaveProfile();
  const me = meQuery.data;

  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (!me || !me.profile) {
    return (
      <ErrorView
        message={errorMessage(meQuery.error, "Your profile could not be loaded.")}
        onRetry={() => void meQuery.refetch()}
      />
    );
  }
  const profile = me.profile;

  const toggle = (field: "isDiscoverable" | "showDistance", value: boolean) => {
    save.mutate(profileToInput(profile, { [field]: value }), {
      onError: (error) => showToast(errorMessage(error, "Could not update this setting."), "error")
    });
  };

  const confirmSignOut = () => {
    Alert.alert("Sign out?", "You can sign back in any time.", [
      { text: "Cancel", style: "cancel" },
      { text: "Sign out", onPress: () => void signOut() }
    ]);
  };

  return (
    <ScreenContainer scroll>
      <Header title="Profile" />
      <View style={styles.identity}>
        <Avatar name={profile.firstName} photo={profile.photos[0]} size={88} />
        <View style={styles.identityText}>
          <Text variant="title">
            {profile.firstName}, {profile.age}
          </Text>
          <Text tone="muted">{profile.locationLabel}</Text>
        </View>
      </View>

      <Section flush title="Your profile">
        <ListItem
          onPress={() => navigation.navigate("EditProfile")}
          right={<Chevron />}
          subtitle="Bio, interests, location"
          title="Edit profile"
        />
        <ListItem
          onPress={() => navigation.navigate("Photos")}
          right={<Chevron />}
          subtitle={`${profile.photos.length} of 6`}
          title="Photos"
        />
        <ListItem
          onPress={() => navigation.navigate("Preferences")}
          right={<Chevron />}
          subtitle={`${me.preferences.minAge} to ${me.preferences.maxAge}, within ${me.preferences.maxDistanceKm} km`}
          title="Who I want to meet"
        />
      </Section>

      <Section flush title="Visibility">
        <ListItem
          right={
            <Switch
              accessibilityLabel="Show my profile in discovery"
              disabled={save.isPending}
              onValueChange={(value) => toggle("isDiscoverable", value)}
              thumbColor={theme.surface}
              trackColor={{ true: theme.primary, false: theme.border }}
              value={profile.isDiscoverable}
            />
          }
          subtitle="Turn off to pause your profile. Your matches can still chat with you."
          title="Discoverable"
        />
        <ListItem
          right={
            <Switch
              accessibilityLabel="Show my distance to others"
              disabled={save.isPending}
              onValueChange={(value) => toggle("showDistance", value)}
              thumbColor={theme.surface}
              trackColor={{ true: theme.primary, false: theme.border }}
              value={profile.showDistance}
            />
          }
          subtitle="Others see an approximate distance, never your location."
          title="Show distance"
        />
      </Section>

      <Section flush title="Safety and account">
        <ListItem onPress={() => navigation.navigate("BlockedUsers")} right={<Chevron />} title="Blocked people" />
        <ListItem onPress={() => navigation.navigate("ChangePassword")} right={<Chevron />} title="Change password" />
        <ListItem onPress={() => navigation.navigate("About")} right={<Chevron />} title="About and privacy" />
        <ListItem onPress={confirmSignOut} title="Sign out" />
        <ListItem onPress={() => navigation.navigate("DeleteAccount")} title="Delete account" tone="danger" />
      </Section>

      <Text center tone="muted" variant="caption">
        Signed in as {me.email}
      </Text>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  identity: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  identityText: { flex: 1, gap: 2 }
});
