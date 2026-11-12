import { z } from 'zod'
import i18n from '../../../shared/i18n'

export function createPassphraseLoginSchema() {
  return z.object({
    passphrase: z.string().min(1, i18n.t('auth:validation.passphraseRequired')),
  })
}

export function createUserLoginSchema() {
  return z.object({
    email: z.string().email(i18n.t('auth:validation.emailInvalid')),
    password: z.string().min(1, i18n.t('auth:validation.passwordRequired')),
  })
}

export function createSignUpSchema() {
  return z
    .object({
      email: z.string().email(i18n.t('auth:validation.emailInvalid')),
      password: z.string().min(8, i18n.t('auth:validation.passwordMin')),
      confirm: z.string().min(1, i18n.t('auth:validation.confirmRequired')),
    })
    .refine((data) => data.password === data.confirm, {
      message: i18n.t('auth:validation.passwordsMismatch'),
      path: ['confirm'],
    })
}

export function createForgotPasswordSchema() {
  return z.object({
    email: z.string().email(i18n.t('auth:validation.emailInvalid')),
  })
}

export function createResetPasswordSchema() {
  return z
    .object({
      password: z.string().min(8, i18n.t('auth:validation.passwordMin')),
      confirm: z.string().min(1, i18n.t('auth:validation.confirmRequired')),
    })
    .refine((data) => data.password === data.confirm, {
      message: i18n.t('auth:validation.passwordsMismatch'),
      path: ['confirm'],
    })
}
