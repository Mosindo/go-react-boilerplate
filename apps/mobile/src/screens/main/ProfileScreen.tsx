import React from "react";
import { Pressable, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { t } from "../../i18n/fr";
import { usePhotos, useProfile } from "../../queries";
import { useSession } from "../../session/SessionProvider";
import { createStyles } from "../../theme/ThemeProvider";
import {
  Button,
  ErrorState,
  FadeIn,
  LinkRow,
  Photo,
  Screen,
  Skeleton,
  Text,
  useDialog
} from "../../ui";
import type { MainStackParams, TabParams } from "../navTypes";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Profile">,
  NativeStackScreenProps<MainStackParams>
>;

const useStyles = createStyles(({ colors, spacing }) => ({
  header: { alignItems: "center", gap: spacing.sm },
  avatar: {
    width: 112,
    height: 112,
    borderRadius: 56,
    borderWidth: 3,
    borderColor: colors.highlight
  },
  links: { alignSelf: "stretch" },
  legal: { minHeight: 44, justifyContent: "center", paddingVertical: spacing.sm }
}));

export default function ProfileScreen({ navigation }: Props) {
  const styles = useStyles();
  const dialog = useDialog();
  const { signOut, me } = useSession();
  const profile = useProfile();
  const photos = usePhotos();
  const main = photos.data?.[0] ?? profile.data?.photos[0];

  const confirmLogout = () =>
    dialog.confirm({
      title: t.profile.logoutConfirmTitle,
      message: t.profile.logoutConfirmText,
      confirmLabel: t.profile.logout,
      destructive: true,
      onConfirm: () => void signOut()
    });

  const showFree = () =>
    dialog.showSheet({
      title: t.profile.freeTitle,
      message: t.profile.freeText,
      actions: [],
      cancelLabel: t.common.close
    });

  return (
    <Screen scroll testID="profile-screen">
      <FadeIn style={styles.header}>
        {profile.isPending ? (
          <>
            <Skeleton height={112} radius={56} width={112} />
            <Skeleton height={24} width={160} />
          </>
        ) : profile.isError ? (
          <ErrorState onRetry={() => profile.refetch()} title={t.profile.loadError} />
        ) : (
          <>
            <Photo name={profile.data.firstName} path={main?.url ?? null} style={styles.avatar} />
            <Text variant="title">
              {profile.data.firstName}, {profile.data.age}
            </Text>
            <Text tone="muted">{profile.data.city}</Text>
          </>
        )}
        {me.data ? (
          <Text tone="muted" variant="caption">
            {me.data.email}
          </Text>
        ) : null}
      </FadeIn>

      <View style={styles.links}>
        <LinkRow
          label={t.profile.edit}
          onPress={() => navigation.navigate("EditProfile")}
          testID="profile-edit-link"
        />
        <LinkRow
          label={t.profile.photos}
          onPress={() => navigation.navigate("ManagePhotos")}
          testID="profile-photos-link"
        />
        <LinkRow
          label={t.profile.preferences}
          onPress={() => navigation.navigate("Preferences")}
          testID="profile-preferences-link"
        />
        <LinkRow
          label={t.profile.privacy}
          onPress={() => navigation.navigate("Privacy")}
          testID="profile-privacy-link"
        />
        <LinkRow
          label={t.profile.blocked}
          onPress={() => navigation.navigate("Blocked")}
          testID="profile-blocked-link"
        />
      </View>

      <Button
        label={t.profile.logout}
        onPress={confirmLogout}
        testID="profile-logout-button"
        variant="secondary"
      />
      <Button
        label={t.profile.deleteAccount}
        onPress={() => navigation.navigate("DeleteAccount")}
        testID="profile-delete-button"
        variant="ghost"
      />

      <Pressable
        accessibilityRole="link"
        accessibilityLabel={t.profile.free}
        onPress={showFree}
        style={styles.legal}
        testID="profile-free-note"
      >
        <Text center style={{ textDecorationLine: "underline" }} tone="muted" variant="caption">
          {t.profile.free}
        </Text>
      </Pressable>
    </Screen>
  );
}
