import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import type { CompositeScreenProps } from '@react-navigation/native';
import type { BottomTabScreenProps } from '@react-navigation/bottom-tabs';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { errorMessage } from '../../api/errors';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import type { MainStackParamList, TabParamList } from '../../navigation/types';
import { useMyProfile, useUnreadNotifications } from '../../query/hooks';
import {
  Avatar,
  ConfirmDialog,
  ErrorState,
  ListRow,
  LoadingView,
  Screen,
  Text,
} from '../../shared/ui';
import { spacing, useTheme } from '../../theme';

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, 'Profile'>,
  NativeStackScreenProps<MainStackParamList>
>;

export function ProfileScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const { data, isPending, isError, error, refetch } = useMyProfile();
  const unread = useUnreadNotifications();
  const [confirmLogout, setConfirmLogout] = useState(false);

  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  const go =
    <K extends keyof MainStackParamList>(name: K) =>
    () =>
      (navigation.navigate as (n: K) => void)(name);

  return (
    <Screen scroll padded={false} contentStyle={styles.content} testID="profile-screen">
      <View style={[styles.hero, { backgroundColor: colors.surface, borderColor: colors.border }]}>
        <Avatar name={data.firstName} path={data.photos[0]?.url} size={84} testID="my-avatar" />
        <View style={styles.heroText}>
          <Text variant="title" testID="my-name">
            {data.firstName}, {data.age}
          </Text>
          <Text tone="muted">{data.city}</Text>
        </View>
      </View>

      <Section title={fr.profile.title}>
        <ListRow
          icon="eye-outline"
          label={fr.profile.preview}
          onPress={go('MyPreview')}
          testID="row-preview"
        />
        <ListRow
          icon="create-outline"
          label={fr.profile.edit}
          onPress={go('EditProfile')}
          testID="row-edit-profile"
        />
        <ListRow
          icon="images-outline"
          label={fr.photos.edit}
          onPress={go('EditPhotos')}
          testID="row-edit-photos"
        />
        <ListRow
          icon="options-outline"
          label={fr.profile.preferences}
          onPress={go('EditPreferences')}
          testID="row-preferences"
        />
        <ListRow
          icon="lock-closed-outline"
          label={fr.profile.privacy}
          onPress={go('Privacy')}
          testID="row-privacy"
        />
      </Section>

      <Section title={fr.profile.myAccount}>
        <ListRow
          icon="notifications-outline"
          label={fr.profile.notifications}
          badge={unread}
          onPress={go('Notifications')}
          testID="row-notifications"
        />
        <ListRow
          icon="ban-outline"
          label={fr.profile.blocked}
          onPress={go('Blocked')}
          testID="row-blocked"
        />
        <ListRow
          icon="key-outline"
          label={fr.profile.changePassword}
          onPress={go('ChangePassword')}
          testID="row-change-password"
        />
        <ListRow
          icon="log-out-outline"
          label={fr.auth.logout}
          hideChevron
          onPress={() => setConfirmLogout(true)}
          testID="row-logout"
        />
        <ListRow
          icon="trash-outline"
          label={fr.profile.deleteAccount}
          destructive
          onPress={go('DeleteAccount')}
          testID="row-delete-account"
        />
      </Section>

      <ConfirmDialog
        visible={confirmLogout}
        title={fr.auth.logoutConfirmTitle}
        body={fr.auth.logoutConfirmBody}
        confirmLabel={fr.auth.logout}
        onCancel={() => setConfirmLogout(false)}
        onConfirm={() => {
          setConfirmLogout(false);
          void signOut();
        }}
      />
    </Screen>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text variant="label" tone="muted" style={styles.sectionTitle} accessibilityRole="header">
        {title}
      </Text>
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg },
  hero: {
    flexDirection: 'row',
    alignItems: 'center',
    padding: spacing.lg,
    borderRadius: 20,
    borderWidth: 1,
  },
  heroText: { marginLeft: spacing.lg, flex: 1 },
  section: { marginTop: spacing.xl },
  sectionTitle: { marginBottom: spacing.xs, textTransform: 'uppercase', letterSpacing: 0.6 },
});
