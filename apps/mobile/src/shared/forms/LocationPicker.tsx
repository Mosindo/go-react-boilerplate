import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import * as Location from 'expo-location';
import { fr } from '../../i18n/fr';
import { spacing, useTheme } from '../../theme';
import { Button, Chip, Text, TextField } from '../ui';
import { CITY_PRESETS } from './cities';

export type LocationValue = { city: string; latitude: number | null; longitude: number | null };

type Status = 'idle' | 'locating' | 'denied' | 'error' | 'ok';

const round2 = (n: number) => Math.round(n * 100) / 100;

export function LocationPicker({
  value,
  onChange,
  cityError,
  coordsError,
}: {
  value: LocationValue;
  onChange: (next: LocationValue) => void;
  cityError?: string | null;
  coordsError?: string | null;
}) {
  const { colors } = useTheme();
  const [status, setStatus] = useState<Status>(value.latitude !== null ? 'ok' : 'idle');
  const hasCoords = value.latitude !== null && value.longitude !== null;

  const locate = async () => {
    setStatus('locating');
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== 'granted') {
        setStatus('denied');
        return;
      }
      const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const latitude = round2(pos.coords.latitude);
      const longitude = round2(pos.coords.longitude);
      let city = '';
      try {
        const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
        city = place?.city ?? place?.subregion ?? place?.district ?? '';
      } catch {
        // reverse geocoding is not available everywhere (e.g. web): the user types the city.
      }
      onChange({ city: city || value.city, latitude, longitude });
      setStatus('ok');
    } catch {
      setStatus('error');
    }
  };

  const message =
    status === 'denied'
      ? fr.onboarding.locationDenied
      : status === 'error'
        ? fr.onboarding.locationUnavailable
        : status === 'ok' && hasCoords
          ? value.city
            ? fr.onboarding.locationOk(value.city)
            : fr.onboarding.locationOkNoCity
          : null;

  return (
    <View>
      <Button
        label={
          status === 'denied' || status === 'error'
            ? fr.onboarding.locationRetry
            : fr.onboarding.useMyLocation
        }
        icon="locate"
        loading={status === 'locating'}
        onPress={locate}
        variant={hasCoords ? 'secondary' : 'primary'}
        testID="location-use"
      />
      {status === 'locating' ? (
        <Text variant="caption" tone="muted" style={styles.msg}>
          {fr.onboarding.locating}
        </Text>
      ) : null}
      {message ? (
        <Text
          variant="caption"
          tone={status === 'ok' ? 'success' : 'danger'}
          accessibilityRole={status === 'ok' ? undefined : 'alert'}
          testID="location-status"
          style={styles.msg}
        >
          {message}
        </Text>
      ) : null}

      <View style={styles.city}>
        <TextField
          testID="onboarding-city"
          label={fr.onboarding.city}
          placeholder={fr.onboarding.cityPlaceholder}
          value={value.city}
          onChangeText={(city) => onChange({ ...value, city })}
          error={cityError}
          autoCapitalize="words"
          maxLength={80}
        />
      </View>

      <Text variant="label" tone="muted" style={styles.presetsTitle}>
        {fr.onboarding.orPickCity}
      </Text>
      <View style={styles.presets}>
        {CITY_PRESETS.map((c) => (
          <Chip
            key={c.id}
            testID={`city-preset-${c.id}`}
            label={c.name}
            selected={hasCoords && value.city === c.name && value.latitude === c.latitude}
            onPress={() => {
              onChange({ city: c.name, latitude: c.latitude, longitude: c.longitude });
              setStatus('ok');
            }}
          />
        ))}
      </View>
      {coordsError ? (
        <Text
          variant="caption"
          style={{ color: colors.danger }}
          accessibilityRole="alert"
          testID="location-coords-error"
        >
          {coordsError}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  msg: { marginTop: spacing.sm },
  city: { marginTop: spacing.xl },
  presetsTitle: { marginBottom: spacing.sm },
  presets: { flexDirection: 'row', flexWrap: 'wrap' },
});
