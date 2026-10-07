import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React from "react";
import { Alert, Image, StyleSheet, View } from "react-native";
import { resolveMediaUrl } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { useMyProfile } from "../../hooks/useProfile";
import { strings } from "../../i18n/strings";
import type { ProfileStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { FormScreen } from "../../shared/layout";
import { Badge, Button, ListItem, Text, radii, spacing } from "../../shared/ui";
import { useTheme } from "../../theme/useTheme";

type Props = NativeStackScreenProps<ProfileStackParamList, "ProfileHome">;

export default function ProfileHomeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { logout, user } = useAuth();
  const query = useMyProfile();
  const profile = query.data;

  function confirmLogout() {
    Alert.alert(strings.profile.logoutTitle, strings.profile.logoutBody, [
      { text: strings.common.cancel, style: "cancel" },
      { text: strings.profile.logout, style: "destructive", onPress: () => void logout() }
    ]);
  }

  if (query.isLoading) {
    return (
      <FormScreen edges={["top", "left", "right"]}>
        <LoadingView fullScreen />
      </FormScreen>
    );
  }
  if (query.isError || !profile) {
    return (
      <FormScreen edges={["top", "left", "right"]}>
        <ErrorView message={strings.errors.generic} onAction={() => void query.refetch()} />
      </FormScreen>
    );
  }

  const main = [...profile.photos].sort((a, b) => a.position - b.position)[0];

  return (
    <FormScreen edges={["top", "left", "right"]}>
      <View style={[styles.hero, { backgroundColor: colors.surfaceSubtle }]}>
        {main ? (
          <Image source={{ uri: resolveMediaUrl(main.url) }} style={styles.heroImage} />
        ) : (
          <View style={styles.heroEmpty}>
            <Text tone="muted">{strings.profile.noPhoto}</Text>
          </View>
        )}
      </View>

      <View style={styles.gap}>
        <Text variant="title" weight="bold">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city ? <Text tone="muted">{profile.city}</Text> : null}
        {user?.email ? (
          <Text tone="muted" variant="caption">
            {user.email}
          </Text>
        ) : null}
        {profile.bio ? <Text>{profile.bio}</Text> : <Text tone="muted">{strings.profile.noBio}</Text>}
        <View style={styles.chips}>
          {profile.interests.map((interest) => (
            <Badge key={interest.slug} label={interest.label} variant="primary" />
          ))}
        </View>
      </View>

      <View style={styles.gap}>
        <ListItem
          onPress={() => navigation.navigate("EditProfile")}
          title={strings.profile.edit}
          trailing={<Chevron />}
        />
        <ListItem
          onPress={() => navigation.navigate("Photos")}
          title={strings.profile.photos}
          trailing={<Chevron />}
        />
        <ListItem
          onPress={() => navigation.navigate("Preferences")}
          title={strings.profile.preferences}
          trailing={<Chevron />}
        />
        <ListItem
          onPress={() => navigation.navigate("BlockedUsers")}
          title={strings.profile.blocked}
          trailing={<Chevron />}
        />
      </View>

      <View style={styles.gap}>
        <Button label={strings.profile.logout} onPress={confirmLogout} variant="outline" />
        <Button
          label={strings.profile.deleteAccount}
          onPress={() => navigation.navigate("DeleteAccount")}
          variant="ghost"
        />
      </View>
    </FormScreen>
  );
}

function Chevron() {
  return (
    <Text tone="muted" variant="heading">
      {">"}
    </Text>
  );
}

const styles = StyleSheet.create({
  hero: { width: "100%", aspectRatio: 1.1, borderRadius: radii.xl, overflow: "hidden" },
  heroImage: { width: "100%", height: "100%" },
  heroEmpty: { flex: 1, alignItems: "center", justifyContent: "center" },
  gap: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm, marginTop: spacing.xs }
});
