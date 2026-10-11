import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ForgotPasswordScreen } from '../features/auth/ForgotPasswordScreen';
import { LoginScreen } from '../features/auth/LoginScreen';
import { RegisterScreen } from '../features/auth/RegisterScreen';
import { ResetPasswordScreen } from '../features/auth/ResetPasswordScreen';
import { WelcomeScreen } from '../features/auth/WelcomeScreen';
import { fr } from '../i18n';
import type { AuthStackParamList } from './types';
import { useStackOptions } from './useStackOptions';

const Stack = createNativeStackNavigator<AuthStackParamList>();

export function AuthNavigator() {
  const options = useStackOptions();
  return (
    <Stack.Navigator screenOptions={options}>
      <Stack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <Stack.Screen name="Login" component={LoginScreen} options={{ title: '' }} />
      <Stack.Screen name="Register" component={RegisterScreen} options={{ title: '' }} />
      <Stack.Screen
        name="ForgotPassword"
        component={ForgotPasswordScreen}
        options={{ title: fr.auth.forgotNav }}
      />
      <Stack.Screen
        name="ResetPassword"
        component={ResetPasswordScreen}
        options={{ title: fr.auth.resetNav }}
      />
    </Stack.Navigator>
  );
}
