import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';
import { BirthDateInput, type BirthDateParts } from '../forms/BirthDateInput';
import { initialStep } from '../../screens/onboarding/OnboardingScreen';
import { Button } from './Button';
import { Checkbox } from './Checkbox';
import { ChoiceGroup, Stepper } from './Rows';
import { TextField } from './TextField';

describe('Button', () => {
  it('calls onPress, exposes role/label and blocks presses while loading', () => {
    const onPress = jest.fn();
    const { rerender } = render(<Button label="Continuer" onPress={onPress} testID="b" />);
    const button = screen.getByRole('button', { name: 'Continuer' });
    fireEvent.press(button);
    expect(onPress).toHaveBeenCalledTimes(1);
    rerender(<Button label="Continuer" onPress={onPress} loading testID="b" />);
    fireEvent.press(screen.getByTestId('b'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });
});

describe('TextField', () => {
  it('shows the error as an alert and toggles password visibility', () => {
    render(
      <TextField
        label="Mot de passe"
        testID="pw"
        value="secret"
        onChangeText={() => undefined}
        secure
        error="Trop court"
      />,
    );
    expect(screen.getByRole('alert')).toHaveTextContent('Trop court');
    const input = screen.getByLabelText('Mot de passe');
    expect(input.props.secureTextEntry).toBe(true);
    fireEvent.press(screen.getByTestId('pw-toggle'));
    expect(screen.getByLabelText('Mot de passe').props.secureTextEntry).toBe(false);
  });
});

describe('Checkbox', () => {
  it('toggles with checked state and shows its error', () => {
    const onChange = jest.fn();
    render(
      <Checkbox
        label="Je confirme avoir 18 ans ou plus"
        checked={false}
        onChange={onChange}
        error="Requis"
        testID="cb"
      />,
    );
    const cb = screen.getByRole('checkbox', { name: 'Je confirme avoir 18 ans ou plus' });
    expect(cb.props.accessibilityState).toMatchObject({ checked: false });
    fireEvent.press(cb);
    expect(onChange).toHaveBeenCalledWith(true);
    expect(screen.getByText('Requis')).toBeTruthy();
  });
});

describe('ChoiceGroup / Stepper', () => {
  it('reports toggled values', () => {
    const onToggle = jest.fn();
    render(
      <ChoiceGroup
        mode="multiple"
        label="Je souhaite rencontrer"
        choices={[
          { value: 'man', label: 'Des hommes' },
          { value: 'woman', label: 'Des femmes' },
        ]}
        selected={['man']}
        onToggle={onToggle}
        testIDPrefix="pref"
      />,
    );
    fireEvent.press(screen.getByTestId('pref-woman'));
    expect(onToggle).toHaveBeenCalledWith('woman');
    expect(screen.getByTestId('pref-man').props.accessibilityState).toMatchObject({
      checked: true,
    });
  });

  it('stays within bounds', () => {
    const onChange = jest.fn();
    render(
      <Stepper testID="age" label="Âge minimum" value={18} min={18} max={30} onChange={onChange} />,
    );
    fireEvent.press(screen.getByTestId('age-minus'));
    expect(onChange).not.toHaveBeenCalled();
    fireEvent.press(screen.getByTestId('age-plus'));
    expect(onChange).toHaveBeenCalledWith(19);
  });
});

describe('BirthDateInput', () => {
  it('keeps digits only and reports parts', () => {
    const onChange = jest.fn();
    const value: BirthDateParts = { day: '', month: '', year: '' };
    render(<BirthDateInput value={value} onChange={onChange} error="Date invalide" />);
    fireEvent.changeText(screen.getByLabelText('Jour'), '1a5');
    expect(onChange).toHaveBeenCalledWith({ day: '15', month: '', year: '' });
    expect(screen.getByRole('alert')).toHaveTextContent('Date invalide');
  });
});

describe('onboarding resume', () => {
  const ob = (profile: boolean, photos: boolean, preferences: boolean) => ({
    profile,
    photos,
    preferences,
    complete: false,
  });
  const profile = { bio: '', interests: [] } as never;
  it('resumes at the first missing step', () => {
    expect(initialStep(ob(false, false, false), undefined)).toBe('basics');
    expect(initialStep(ob(true, false, false), profile)).toBe('photos');
    expect(initialStep(ob(true, true, false), profile)).toBe('bio');
    expect(initialStep(ob(true, true, false), { bio: 'hi', interests: [] } as never)).toBe(
      'preferences',
    );
  });
});
