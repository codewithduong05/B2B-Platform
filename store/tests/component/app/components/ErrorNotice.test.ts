// @vitest-environment happy-dom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ErrorNotice from '~/components/ErrorNotice.vue'
import type { AppError } from '#shared/errors'

describe('ErrorNotice', () => {
  const baseError: AppError = {
    code: 'unauthorized',
    message: 'Active session is required.',
    requestId: 'req_abc123',
  }

  it('renders error code and requestId', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    expect(wrapper.text()).toContain('unauthorized')
    expect(wrapper.text()).toContain('req_abc123')
  })

  it('renders error message', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    expect(wrapper.text()).toContain('Active session is required.')
  })

  it('renders Something went wrong heading', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    expect(wrapper.find('h2').text()).toBe('Something went wrong')
  })

  it('has role=alert and aria-live=assertive', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    const section = wrapper.find('section')
    expect(section.attributes('role')).toBe('alert')
    expect(section.attributes('aria-live')).toBe('assertive')
  })

  it('does not render fields list when error has no fields', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    expect(wrapper.find('.notice-fields').exists()).toBe(false)
  })

  it('renders fields list when error has fields', () => {
    const errorWithFields: AppError = {
      ...baseError,
      fields: { email: 'Required', name: 'Too short' },
    }
    const wrapper = mount(ErrorNotice, { props: { error: errorWithFields } })
    const fields = wrapper.findAll('.notice-field')
    expect(fields).toHaveLength(2)
    expect(fields[0].text()).toContain('email')
    expect(fields[0].text()).toContain('Required')
    expect(fields[1].text()).toContain('name')
    expect(fields[1].text()).toContain('Too short')
  })

  it('has correct CSS classes', () => {
    const wrapper = mount(ErrorNotice, { props: { error: baseError } })
    const section = wrapper.find('section')
    expect(section.classes()).toContain('notice')
    expect(section.classes()).toContain('notice-error')
  })
})
