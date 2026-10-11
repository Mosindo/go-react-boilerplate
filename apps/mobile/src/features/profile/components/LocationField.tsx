import { useState } from 'react';
import { View } from 'react-native';
import { useMutation } from '@tanstack/react-query';
import { profileApi } from '../../../api';
import { fr } from '../../../i18n';
import { getApproxLocation } from '../../../lib/location';
import { useErrorToast } from '../../../lib/useErrorToast';
import { useStyles } from '../../../theme';
import { Button, Card, Input, Text, useToast } from '../../../ui';

interface LocationFieldProps {
  city: string;
  onCityChange: (city: string) => void;
  cityError?: string | null;
  /** True when the server already holds a location for this user. */
  hasLocation: boolean;
  /**
   * `deferred`: coordinates are handed to the parent (onboarding saves them after the profile exists).
   * `immediate`: PUT /me/location right away.
   */
  mode: 'deferred' | 'immediate';
  coordinates?: { latitude: number; longitude: number } | null;
  onCoordinates?: (coords: { latitude: number; longitude: number }) => void;
  onSaved?: () => void;
}

export function LocationField({
  city,
  onCityChange,
  cityError,
  hasLocation,
  mode,
  coordinates,
  onCoordinates,
  onSaved,
}: LocationFieldProps) {
  const toast = useToast();
  const onError = useErrorToast();
  const [busy, setBusy] = useState(false);
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.md },
    explain: { gap: t.spacing.sm },
  }));

  const save = useMutation({
    mutationFn: (coords: { latitude: number; longitude: number }) => profileApi.setLocation(coords),
    onSuccess: () => {
      toast.success(fr.location.saved);
      onSaved?.();
    },
    onError,
  });

  const locate = async () => {
    setBusy(true);
    const result = await getApproxLocation();
    setBusy(false);
    if (!result.ok) {
      toast.error(result.reason === 'denied' ? fr.location.denied : fr.location.unavailable);
      return;
    }
    const coords = { latitude: result.latitude, longitude: result.longitude };
    if (mode === 'deferred') {
      onCoordinates?.(coords);
      toast.success(fr.location.ready);
    } else {
      save.mutate(coords);
    }
  };

  const known = hasLocation || !!coordinates;
  return (
    <View style={styles.wrap}>
      <Input
        label={fr.profile.city}
        value={city}
        onChangeText={onCityChange}
        error={cityError}
        maxLength={80}
        autoCapitalize="words"
        autoComplete="postal-address-locality"
        placeholder={fr.profile.cityPlaceholder}
        testID="city"
      />
      <Card style={styles.explain}>
        <Text weight="600">{fr.location.title}</Text>
        <Text variant="small" color="textMuted">
          {fr.location.explain}
        </Text>
        {known ? (
          <Text variant="small" color="success" weight="600">
            {fr.location.known}
          </Text>
        ) : null}
        <Button
          label={known ? fr.location.refresh : fr.location.use}
          variant="soft"
          size="sm"
          loading={busy || save.isPending}
          onPress={() => void locate()}
        />
      </Card>
    </View>
  );
}
