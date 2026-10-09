import React from 'react';
import { StyleSheet } from 'react-native';
import { errorMessage } from '../../api/errors';
import { useMyProfile } from '../../query/hooks';
import { ProfileView } from '../../shared/ProfileView';
import { ErrorState, LoadingView, Screen } from '../../shared/ui';
import { spacing } from '../../theme';

export function MyPreviewScreen() {
  const { data, isPending, isError, error, refetch } = useMyProfile();
  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;
  return (
    <Screen scroll padded={false} contentStyle={styles.content} testID="preview-screen">
      <ProfileView profile={data} />
    </Screen>
  );
}

const styles = StyleSheet.create({ content: { padding: spacing.lg } });
