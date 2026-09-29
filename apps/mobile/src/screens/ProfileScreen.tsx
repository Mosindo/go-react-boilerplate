import React, { useState } from "react";
import { View } from "react-native";
import { useAuth } from "../hooks/useAuth";
import { useMyProfile } from "../hooks/useProfileData";
import { messageFromError } from "../lib/errors";
import { ErrorView } from "../shared/feedback/ErrorView";
import { LoadingView } from "../shared/feedback/LoadingView";
import { KeyboardScreen } from "../shared/layout/KeyboardScreen";
import { Button } from "../shared/ui/Button";
import { ListItem } from "../shared/ui/ListItem";
import { Text } from "../shared/ui/Text";
import { type Theme } from "../shared/ui/theme";
import { useThemedStyles } from "../shared/ui/useThemedStyles";
import { ProfileEditScreen } from "./ProfileEditScreen";
import { ProfilePhotosScreen } from "./ProfilePhotosScreen";
import { ProfilePreferencesScreen } from "./ProfilePreferencesScreen";
import { ProfilePreviewCard } from "./ProfilePreviewCard";
import { ProfilePrivacyScreen } from "./ProfilePrivacyScreen";
import { APP_VERSION, SettingsAboutScreen } from "./SettingsAboutScreen";
import { SettingsBlockedScreen } from "./SettingsBlockedScreen";
import { SettingsDeleteScreen } from "./SettingsDeleteScreen";
import { SettingsPasswordScreen } from "./SettingsPasswordScreen";

type ProfileRoute =
  | "home"
  | "edit"
  | "photos"
  | "preferences"
  | "privacy"
  | "blocked"
  | "password"
  | "about"
  | "delete";

const makeStyles = (t: Theme) => ({
  menu: { gap: t.spacing.sm },
  footer: { alignItems: "center" as const, gap: t.spacing.sm }
});

/** Profile tab: own profile preview plus settings. */
export default function ProfileScreen() {
  const [route, setRoute] = useState<ProfileRoute>("home");
  const back = () => setRoute("home");

  switch (route) {
    case "edit":
      return <ProfileEditScreen onBack={back} />;
    case "photos":
      return <ProfilePhotosScreen onBack={back} />;
    case "preferences":
      return <ProfilePreferencesScreen onBack={back} />;
    case "privacy":
      return <ProfilePrivacyScreen onBack={back} />;
    case "blocked":
      return <SettingsBlockedScreen onBack={back} />;
    case "password":
      return <SettingsPasswordScreen onBack={back} />;
    case "about":
      return <SettingsAboutScreen onBack={back} />;
    case "delete":
      return <SettingsDeleteScreen onBack={back} />;
    default:
      return <ProfileHome onNavigate={setRoute} />;
  }
}

function ProfileHome({ onNavigate }: { onNavigate: (route: ProfileRoute) => void }) {
  const styles = useThemedStyles(makeStyles);
  const { signOut, user } = useAuth();
  const query = useMyProfile();
  const [signingOut, setSigningOut] = useState(false);

  async function handleSignOut() {
    setSigningOut(true);
    try {
      await signOut();
    } finally {
      setSigningOut(false);
    }
  }

  return (
    <KeyboardScreen edges={["top", "left", "right"]} testID="profile-screen">
      <Text accessibilityRole="header" variant="title" weight="bold">
        Profile
      </Text>

      {query.isPending ? <LoadingView label="Loading your profile..." /> : null}
      {query.isError && query.data === undefined ? (
        <ErrorView message={messageFromError(query.error)} onAction={() => void query.refetch()} />
      ) : null}
      {query.data ? (
        <>
          <Text tone="muted" variant="caption">
            This is how other people see you.
          </Text>
          <ProfilePreviewCard profile={query.data} />
        </>
      ) : null}

      <View style={styles.menu}>
        <ListItem onPress={() => onNavigate("edit")} subtitle="Name, bio, city, interests" testID="menu-edit" title="Edit profile" />
        <ListItem onPress={() => onNavigate("photos")} subtitle="Add, remove, reorder" testID="menu-photos" title="Photos" />
        <ListItem onPress={() => onNavigate("preferences")} subtitle="Who you see, age, distance, location" testID="menu-preferences" title="Discovery preferences" />
        <ListItem onPress={() => onNavigate("privacy")} subtitle="Pause your profile, hide age or distance" testID="menu-privacy" title="Privacy" />
        <ListItem onPress={() => onNavigate("blocked")} testID="menu-blocked" title="Blocked people" />
        <ListItem onPress={() => onNavigate("password")} testID="menu-password" title="Change password" />
        <ListItem onPress={() => onNavigate("about")} subtitle="Safety tips and app info" testID="menu-about" title="About and safety" />
      </View>

      <View style={styles.footer}>
        <Text tone="muted" variant="caption">
          {user?.email ?? ""}
        </Text>
        <Button
          label="Sign out"
          loading={signingOut}
          onPress={() => void handleSignOut()}
          testID="profile-signout"
          variant="outline"
        />
        <Button
          label="Delete account"
          onPress={() => onNavigate("delete")}
          testID="menu-delete"
          variant="ghost"
        />
        <Text tone="subtle" variant="caption">
          Alba {APP_VERSION}
        </Text>
      </View>
    </KeyboardScreen>
  );
}
